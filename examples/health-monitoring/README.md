# Health Monitoring System

A practical example demonstrating concurrent health checks for microservices, similar to systems used by Netflix, Uber, and other companies with service mesh architectures.

## Features Demonstrated

- **Concurrent Execution**: Multiple health checks running simultaneously
- **Error Collection**: Using `CollectAll` strategy to check all services even if some fail
- **Timeout Management**: 5-second timeout for health checks
- **Real-world Patterns**: Service discovery, health aggregation, alerting

## Industry Application

This pattern is used by:
- **Netflix**: Service mesh health monitoring
- **Uber**: Microservices health checks
- **Airbnb**: Service availability monitoring
- **Any company** with microservices architecture

## Running the Example

```bash
cd examples/health-monitoring
go run main.go
```

## Sample Output

```
🏥 Health Monitoring System - Service Mesh Health Checks
Real-world example: Netflix/Uber service monitoring

   🔍 Checking user-service health...
   🔍 Checking payment-service health...
   🔍 Checking inventory-service health...
   🔍 Checking notification-service health...
   🔍 Checking analytics-service health...
   ✅ notification-service: HEALTHY (52ms)
   ✅ analytics-service: HEALTHY (89ms)
   ❌ payment-service: Connection timeout to http://payment-service:8081/health
   ✅ user-service: HEALTHY (156ms)
   ✅ inventory-service: DEGRADED (198ms)

✅ Health check completed in 200ms

📊 Health Check Report
==================================================
Overall Status: ⚠️  DEGRADED
Healthy Services: 3/5
Check Time: 2025-08-27 11:54:32

Service Details:
  ✅ HEALTHY          user-service         | HEALTHY  |   156ms
  ❌ DOWN             payment-service      | DOWN     |    52ms | Connection timeout to http://payment-service:8081/health
  ⚠️  DEGRADED        inventory-service    | DEGRADED |   198ms
  ✅ HEALTHY          notification-service | HEALTHY  |    52ms
  ✅ HEALTHY          analytics-service    | HEALTHY  |    89ms

🚨 ALERT TRIGGERED
==============================
System Status: DEGRADED
Affected Services: 2
📧 Notifications sent to:
  - DevOps team via Slack
  - On-call engineer via PagerDuty
  - Monitoring dashboard updated
```

## Key Learning Points

1. **Concurrent Health Checks**: All services are checked simultaneously for faster results
2. **Error Handling**: Failed health checks don't stop other checks from completing
3. **Comprehensive Reporting**: Aggregated view of system health
4. **Alerting Integration**: Automatic notifications based on health status
5. **Production Patterns**: Realistic service monitoring implementation

## Real-world Extensions

In production, this would typically include:
- Integration with service discovery (Consul, etcd)
- Metrics collection (Prometheus, DataDog)
- Dashboard updates (Grafana, custom dashboards)
- Auto-scaling triggers based on health status
- Circuit breaker pattern integration