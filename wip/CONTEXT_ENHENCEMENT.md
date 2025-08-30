# Context Enhancements

This document reviews the enhanced orchestration Context and aligns it with Go's context best practices. It highlights what's already solid, where it diverges from idioms, and provides concrete recommendations with examples, plus Do's and Don'ts for everyday use.

## What's good in the current design

- Uses the standard library for cancellation and deadlines (context.WithCancel/WithDeadline), and exposes Done() and Err().
- Child derivation (CreateChild) correctly chains to the parent's context so cancellation propagates.
- Thread-safe storage and timer handling with mutexes/atomics; zero-allocation hot-path helpers like IsExpired() and GetRemainingTime().
- Useful orchestration ergonomics: hierarchical paths (GetPath), parent/child links, and a dedicated graceful shutdown signal separate from cancel.
- Clear separation of concerns for timeout helpers vs. shutdown coordination.

## Gaps vs. Go idioms (and why they matter)

- Doesn't implement context.Context. Most Go APIs accept context.Context; without Deadline() and Value(), you lose plug‑and‑play interoperability and tracing tools.
- Exposes Cancel() on the read interface. In idiomatic Go, only the creator should cancel; handing out a cancel method invites accidental global cancels.
- Mutable key/value map with string keys on the context. This diverges from std context.Value guidance on purpose for orchestration needs. That's fine when treated as an orchestration-local state store with clear guardrails (see "Using mutable orchestration state safely").
- Redundant timeout timer alongside context.WithDeadline. Duplicating timers risks double work and leaks; std context cancellation already handles deadlines.
- Value inheritance mismatch. Comments suggest children inherit values, but code initializes a fresh map and lookups don't fall back to the parent. This is by design; there is no implicit parent read‑through.
- No cancel causes. With Go 1.20+, context.WithCancelCause and context.Cause allow precise error semantics (timeout vs. business cancel).

## Recommended design adjustments

1. Implement context.Context on the enhanced Context

- Add Deadline() (time.Time, bool) and Value(key any) any; keep Done() and Err().
- Optionally expose Std() context.Context to access the underlying context when needed.

1. Split reader vs controller roles

- Constructors should return (Context, CancelFunc) and only the owner holds the cancel func.
- Alternatively, define two interfaces: Context (read-only API) and Controller (Cancel(), InitiateShutdown()). Public APIs accept Context only.

1. Prefer std deadline mechanics, simplify timers

- Rely on context.WithDeadline/WithTimeout to trigger cancellation.
- Keep IsExpired/GetRemainingTime by consulting ctx.Deadline() and time.Until(deadline) (atomic cache optional).

1. Rework values to be idiomatic and safe

- For cross-cutting data, prefer ctx.Value with typed keys (unexported types), immutable semantics.
- If you need a mutable state bag for orchestration, expose it separately (e.g., State() or Store) rather than as context values; document that it doesn't auto‑inherit unless explicitly copied.
- If you keep inheritance semantics, implement hierarchical read‑through to parent and document write locality.

1. Add cancel causes and reasoned cancellation

- Use context.WithCancelCause internally and expose CancelWithReason(err error).
- Document that Err() reflects the cause via context.Cause(ctx).

1. Clarify shutdown vs cancel semantics

- Cancel(): immediate abort; Shutdown(): graceful wind‑down signal. Document that they are independent channels.
- Provide InitiateShutdown() only to the owner/controller.

<!-- Consolidated in the Do's and Don'ts section near the end -->

## Interoperability and cancel cause

```go
// Deadline and Value forward to std context for interop
func (c *contextImpl) Deadline() (time.Time, bool) { return c.ctx.Deadline() }
func (c *contextImpl) Value(key any) any          { return c.ctx.Value(key) }

// Constructor returns reader + cancel func; only owner can cancel
func NewFromStd(parent context.Context) (Context, context.CancelFunc) {
	// WithCancelCause in Go 1.20+
	ctx, cancel := context.WithCancelCause(parent)
	c := &contextImpl{ /* init fields, ctx: ctx */ }
	// return a cancel without exposing it on the read-only interface
	cancelFn := func() { context.CancelCause(c.ctx, context.Canceled) }
	_ = cancel // kept if owner needs direct cancel
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
- Update NewContext/WithDeadline/WithTimeout to return (Context, CancelFunc) or keep Cancel() but don't expose it on the public interface; provide a separate Controller interface as needed.
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

Do's

- Use Set/Get for orchestration-local, small, transient state; prefer typed helpers and key namespacing.
- Pass immutable cross-cutting metadata via ctx.Value with typed keys when it must flow through third-party APIs.

Don'ts

- Don't store large or long-lived data in the KV; use external stores and keep references only.
- Don't expose raw string keys widely; centralize them or wrap with helpers to avoid collisions.

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

## The Entrypoint: `Run(ctx context.Context, ...)`

`Run` should accept the standard `context.Context` so callers can pass any context they have (with deadlines/cancellation or already your enhanced Context). Normalize it at the boundary and then use your enhanced features internally.

```go
// In your orchestrator package...

type Orchestrator struct {
	policy Policy
	// other configured dependencies
}

func (o *Orchestrator) Run(ctx context.Context, wf Workflow) (*result.Result, error) {
	// 1) Normalize: support both std context and your Context
	orchCtx := ourcontext.FromStd(ctx)

	// 2) Apply per-step policy timeouts by deriving from current context
	stepCtx, cancel := context.WithTimeout(orchCtx, o.policy.Timeout)
	defer cancel()

	if err := doStep(stepCtx); err != nil {
		return nil, err
	}

	// 3) Third-party calls accept orchCtx directly if your Context implements context.Context
	// db.QueryContext(orchCtx, query, args...)

	// return a result (illustrative)
	return &result.Result{}, nil
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
	// If Context implements context.Context, pass it directly
	stepCtx, cancel := context.WithTimeout(c, policy.Timeout)
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

## Do's and Don'ts

Do

- Accept std context.Context at boundaries; adapt once via FromStd and pass your Context internally.
- Implement context.Context (preferred) or provide Std() for third-party API calls.
- Keep timeouts/retries in policy; derive per-step child contexts at use-sites.
- Use CreateChild("name") to scope state and tracing context.
- Use Shutdown() to coordinate graceful stop; keep Cancel() for immediate abort paths.
- Use typed keys for std ctx metadata; keep orchestration KV entries small and namespaced.

Don't

- Don't seed or override the caller's context from config.
- Don't expose Cancel() on the consumer-facing Context; only owners control cancellation.
- Don't attach large or long-lived data to the KV; store references instead.
- Don't create a parallel timeout timer when using context.WithDeadline/WithTimeout.
- Don't assume implicit value inheritance from parent; there is no read-through by design.

