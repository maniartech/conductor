# Context Enhancements

This document reviews the enhanced orchestration Context and aligns it with Go's context best practices. It highlights what’s already solid, where it diverges from idioms, and provides concrete recommendations with examples, plus Do’s and Don’ts for everyday use.

## What’s good in the current design

- Uses the standard library for cancellation and deadlines (context.WithCancel/WithDeadline), and exposes Done() and Err().
- Child derivation (CreateChild) correctly chains to the parent’s context so cancellation propagates.
- Thread-safe storage and timer handling with mutexes/atomics; zero-allocation hot-path helpers like IsExpired() and GetRemainingTime().
- Useful orchestration ergonomics: hierarchical paths (GetPath), parent/child links, and a dedicated graceful shutdown signal separate from cancel.
- Clear separation of concerns for timeout helpers vs. shutdown coordination.

## Gaps vs. Go idioms (and why they matter)

- Doesn’t implement context.Context. Most Go APIs accept context.Context; without Deadline() and Value(), you lose plug‑and‑play interoperability and tracing tools.
- Exposes Cancel() on the read interface. In idiomatic Go, only the creator should cancel; handing out a cancel method invites accidental global cancels.
- Mutable key/value map with string keys on the context. This diverges from std context.Value guidance on purpose for orchestration needs. That’s fine when treated as an orchestration-local state store with clear guardrails (see “Using mutable orchestration state safely”).
- Redundant timeout timer alongside context.WithDeadline. Duplicating timers risks double work and leaks; std context cancellation already handles deadlines.
- Value inheritance mismatch. Comments suggest children inherit values, but code initializes a fresh map and lookups don’t fall back to the parent.
- No cancel causes. With Go 1.20+, context.WithCancelCause and context.Cause allow precise error semantics (timeout vs. business cancel).

## Recommended design adjustments

1) Implement context.Context on the enhanced Context

- Add Deadline() (time.Time, bool) and Value(key any) any; keep Done() and Err().
- Optionally expose Std() context.Context to access the underlying context when needed.

1) Split reader vs controller roles

- Constructors should return (Context, CancelFunc) and only the owner holds the cancel func.
- Alternatively, define two interfaces: Context (read-only API) and Controller (Cancel(), InitiateShutdown()). Public APIs accept Context only.

1) Prefer std deadline mechanics, simplify timers

- Rely on context.WithDeadline/WithTimeout to trigger cancellation.
- Keep IsExpired/GetRemainingTime by consulting ctx.Deadline() and time.Until(deadline) (atomic cache optional).

1) Rework values to be idiomatic and safe

- For cross-cutting data, prefer ctx.Value with typed keys (unexported types), immutable semantics.
- If you need a mutable state bag for orchestration, expose it separately (e.g., State() or Store) rather than as context values; document that it doesn’t auto‑inherit unless explicitly copied.
- If you keep inheritance semantics, implement hierarchical read‑through to parent and document write locality.

1) Add cancel causes and reasoned cancellation

- Use context.WithCancelCause internally and expose CancelWithReason(err error).
- Document that Err() reflects the cause via context.Cause(ctx).

1) Clarify shutdown vs cancel semantics

- Cancel(): immediate abort; Shutdown(): graceful wind‑down signal. Document that they are independent channels.
- Provide InitiateShutdown() only to the owner/controller.

## Do’s and Don’ts

Do

- Accept the read-only Context interface in all public APIs; don’t require a cancel‑capable type from callers.
- Derive child contexts for sub-orchestrations with CreateChild("name").
- Use WithTimeout/WithDeadline on the parent to bound operations; check ctx.Err() and select on ctx.Done().
- Use typed keys for ctx.Value when attaching metadata (trace IDs, request IDs). Keep values immutable.
- Use Shutdown() to coordinate graceful stop; keep Cancel() for immediate abort paths.

Don’t

- Don’t pass a context that exposes Cancel() to downstream libraries; only the creator should cancel.
- Don’t store large objects or long‑lived mutable state in context values. Prefer a separate state store.
- Don’t create your own parallel timeout timer when using context.WithDeadline; rely on std cancellation.
- Don’t assume child contexts inherit values unless implemented; either copy explicitly or read through parent.

## Example: implementing context.Context and cancel cause

```go
// Deadline and Value forward to std context for interop
func (c *contextImpl) Deadline() (time.Time, bool) { return c.ctx.Deadline() }
func (c *contextImpl) Value(key any) any          { return c.ctx.Value(key) }

// Constructor returns reader + cancel func; only owner can cancel
func NewContext(cfg config.Config) (Context, context.CancelFunc) {
	base := cfg.Context
	if base == nil { base = context.Background() }
	// WithCancelCause in Go 1.20+
	ctx, cancel := context.WithCancelCause(base)
	c := &contextImpl{ /* init fields, ctx: ctx */ }
	// wrap a cancel func without exposing on interface
	cancelFn := func() { context.CancelCause(c.ctx, context.Canceled) }
	_ = cancel // keep if you also need it internally
	return c, cancelFn
}

// Reasoned cancellation
func (c *contextImpl) CancelWithReason(err error) { context.CancelCause(c.ctx, err) }
func (c *contextImpl) Err() error                 { return context.Cause(c.ctx) }
```

## Example: typed keys instead of stringly Get/Set

```go
// Define an unexported key type to avoid collisions
type userIDKey struct{}

func WithUserID(ctx context.Context, id int64) context.Context { return context.WithValue(ctx, userIDKey{}, id) }
func UserIDFrom(ctx context.Context) (int64, bool) {
	v := ctx.Value(userIDKey{})
	id, ok := v.(int64)
	return id, ok
}
```

## Example: graceful shutdown vs cancel

```go
// Somewhere in your orchestrator owner
ctx, cancel := NewContext(cfg)
defer cancel()

// Initiate graceful shutdown from a signal handler
go func() { <-sigCh; owner.InitiateShutdown() }()

// In workers: prefer shutdown for winding down
select {
case <-ctx.Shutdown():
	return nil // finish quickly
case <-ctx.Done():
	return ctx.Err() // immediate abort
}
```

## Migration notes for the current code

- Add Deadline() and Value() methods delegating to c.ctx to implement context.Context.
- Replace internal time.AfterFunc timer with std deadline cancellation; keep IsExpired/GetRemainingTime by reading ctx.Deadline().
- Update NewContext/WithDeadline/WithTimeout to return (Context, CancelFunc) or keep Cancel() but don’t expose it on the public interface; provide a separate Controller interface as needed.
- Decide on value semantics: either remove Get/Set or add parent read‑through and typed helpers; avoid string keys in examples.
- Introduce CancelWithReason and Err() powered by context.WithCancelCause/context.Cause.
- Ensure config inheritance is copy‑safe (no shared mutation) and document it.

---

By adopting these adjustments, the enhanced Context stays ergonomic for orchestration while remaining idiomatic and interoperable with the broader Go ecosystem.

## Using mutable orchestration state safely (by design)

Rationale

- Orchestrations often need to share small, evolving bits of state (intermediate results, flags) across child flows. A mutable KV store is pragmatic for this.
- This KV is not a general replacement for std context.Value; treat it as orchestration-local state with explicit conventions.

Guardrails

- Namespacing: prefix keys per workflow/path (e.g., "pay.authorize.result"). Consider helper functions to build keys from ctx.GetPath().
- Typed accessors: wrap Get/Set with typed helpers to avoid type assertions littered across code.
- Size discipline: store small values only (IDs, small structs). For large blobs, keep them in external storage and store references.
- Concurrency: when multiple goroutines touch the same keys, prefer write-once or use higher-level synchronization to avoid read‑modify‑write races.
- Lifecycle: clean up ephemeral keys or keep them scoped to a child context via CreateChild to limit visibility.

Example: typed helper over Set/Get

```go
const keyAuthResult = "pay.authorize.result"

type AuthResult struct {
	TxID string
	Approved bool
}

func SetAuthResult(ctx context.Context, c orchestration.Context, r AuthResult) {
	// store small struct; callers own larger payloads elsewhere
	c.Set(keyAuthResult, r)
}

func GetAuthResult(c orchestration.Context) (AuthResult, bool) {
	v := c.Get(keyAuthResult)
	r, ok := v.(AuthResult)
	return r, ok
}
```

Example: namespacing with path

```go
func ns(c orchestration.Context, local string) string {
	p := c.GetPath()
	if p == "" { return local }
	return strings.TrimPrefix(p, "/") + "." + local
}

// usage
c.Set(ns(c, "inventory.reserved"), true)
```

Do’s

- Use Set/Get for orchestration-local, small, transient state; prefer typed helpers and key namespacing.
- Pass immutable cross-cutting metadata via ctx.Value with typed keys when it must flow through third-party APIs.

Don’ts

- Don’t store large or long-lived data in the KV; use external stores and keep references only.
- Don’t expose raw string keys widely; centralize them or wrap with helpers to avoid collisions.

## Example: adapting std context at the boundary

```go
// FromStd adapts an incoming std context to your orchestration Context.
// Fast-path: if incoming is already your Context, just return it.
func FromStd(ctx context.Context) Context {
	if c, ok := ctx.(Context); ok {
		return c
	}
	// wrap preserves Done/Err/Deadline/Value behavior; no config seeding here
	return newWrappedContext(ctx)
}
```

## Example: embed vs Std() interop

```go
// Option A: implement context.Context on your type
func (c *contextImpl) Deadline() (time.Time, bool) { return c.ctx.Deadline() }
func (c *contextImpl) Done() <-chan struct{}      { return c.ctx.Done() }
func (c *contextImpl) Err() error                 { return c.ctx.Err() }
func (c *contextImpl) Value(key any) any          { return c.ctx.Value(key) }

// Option B: expose the std ctx explicitly
func (c *contextImpl) Std() context.Context { return c.ctx }

// Usage with third-party APIs
db.QueryContext(c /* or c.Std() */, query, args...)
```

## Example: per-step timeout from policy (not global)

```go
func runStep(c Context, policy Policy) error {
	// derive from current std ctx, respecting caller’s deadline
	std := c.(interface{ Std() context.Context }).Std()
	stepCtx, cancel := context.WithTimeout(std, policy.Timeout)
	defer cancel()
	return callExternal(stepCtx)
}
```

## Example: echo-style handler using your Context

```go
type Handler struct{ Svc *Service }

// Public APIs use your Context
func (h *Handler) Handle(c Context) error {
	// KV: small, namespaced, typed helpers
	SetAuthResult(c, AuthResult{TxID: "t1", Approved: true})

	// Third-party: pass std ctx seamlessly
	if err := h.Svc.Do(c /* implements context.Context */); err != nil {
		return err
	}
	return nil
}
```

## Top-level Do’s and Don’ts (extended)

Do

- Accept std context.Context at boundaries; adapt once via FromStd and pass your Context internally.
- Keep timeouts/retries in policy; derive per-step child contexts at use-sites.
- Implement context.Context or provide Std() for third-party API calls.
- Split roles: read-only Context for consumers; owner-only controller holds cancel/shutdown.

Don’t

- Don’t seed or override the caller’s context from config.
- Don’t attach large or long-lived data to the KV; store references instead.
- Don’t expose Cancel() on the consumer-facing Context.

