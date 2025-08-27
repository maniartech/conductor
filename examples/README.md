# Orchestrator Examples

This directory contains practical, industry-based examples that progressively showcase the full potential of the orchestrator library. Each example is based on real-world scenarios that developers encounter in production environments.

## 🚀 Getting Started

Start with the `hello/` example to understand basic concurrent task execution, then progress through the examples to learn advanced orchestration patterns.

```bash
# Run the hello example
cd examples/hello
go run main.go
```

## 📚 Example Progression

The examples are designed to build upon each other, introducing new concepts and features progressively:

### 1. **Hello World** (`hello/`)
**Status**: ✅ **Available**  
**Industry**: Getting Started  
**Features**: Basic concurrent execution  
**Learn**: Task creation, concurrent execution, result collection

```go
orchestrator.Setup(
    orchestrator.Concurrent(
        orchestrator.Task(fetchUserData).Named("fetch-user"),
        orchestrator.Task(processAnalytics).Named("process-analytics"),
        orchestrator.Task(sendNotifications).Named("send-notifications"),
    ).Named("concurrent-tasks"),
).Await()
```

### 2. **API Health Check System** (`health-monitoring/`)
**Status**: ✅ **Available**  
**Industry**: DevOps/SRE - Service monitoring and alerting  
**Features**: Error handling, timeouts, service monitoring  
**Real-world**: Netflix, Uber service mesh monitoring  
**Learn**: Error strategies, timeout handling, health check patterns

### 3. **E-commerce Order Processing** (`order-processing/`)
**Status**: ✅ **Available**  
**Industry**: E-commerce - Order fulfillment pipeline  
**Features**: Sequential workflows, error boundaries, retries  
**Real-world**: Amazon/Shopify order processing  
**Learn**: Sequential orchestration, critical path processing, retry mechanisms

### 4. **Social Media Content Moderation** (`content-moderation/`)
**Status**: ✅ **Available**  
**Industry**: Social Media - Content safety and compliance  
**Features**: Mixed sequential + concurrent, conditional flows  
**Real-world**: Facebook/Twitter content moderation  
**Learn**: Hybrid orchestration, conditional execution, content processing pipelines

### 5. **Financial Transaction Processing** (`financial-transactions/`)
**Status**: ✅ **Available**  
**Industry**: FinTech - Payment processing and fraud detection  
**Features**: Conditional orchestration, error recovery, audit trails  
**Real-world**: Stripe/PayPal transaction processing  
**Learn**: Risk-based workflows, compliance patterns, financial data handling

### 6. **Video Streaming Pipeline** (`video-processing/`)
**Status**: ✅ **Available**  
**Industry**: Media/Entertainment - Video transcoding and CDN distribution  
**Features**: Progress tracking, resource management, parallel processing  
**Real-world**: YouTube/Netflix video processing  
**Learn**: Progress monitoring, resource-intensive tasks, media processing

### 7. **IoT Data Processing Platform** (`iot-data-processing/`)
**Status**: ✅ **Available**  
**Industry**: IoT/Manufacturing - Sensor data aggregation and analysis  
**Features**: Resource pooling, batch processing, real-time analytics  
**Real-world**: GE Predix, AWS IoT Core  
**Learn**: High-volume data processing, sensor analytics, real-time systems

### 8. **CI/CD Deployment Pipeline** (`cicd-pipeline/`)
**Status**: ✅ **Available**  
**Industry**: DevOps - Continuous integration and deployment  
**Features**: Complex orchestration, environment management, rollback strategies  
**Real-world**: GitHub Actions, GitLab CI, Jenkins  
**Learn**: Deployment automation, testing pipelines, rollback mechanisms

### 9. **Real-time Fraud Detection** (`fraud-detection/`)
**Status**: ✅ **Available**  
**Industry**: Banking/Insurance - Real-time transaction monitoring  
**Features**: Event-driven processing, machine learning integration, alerting  
**Real-world**: Mastercard/Visa fraud detection  
**Learn**: Real-time processing, ML integration, alerting systems

### 10. **Supply Chain Management** (`supply-chain/`)
**Status**: ✅ **Available**  
**Industry**: Logistics/Manufacturing - Inventory and supply optimization  
**Features**: Complex dependencies, external API integration, optimization algorithms  
**Real-world**: Amazon supply chain, Walmart inventory  
**Learn**: Complex workflows, external integrations, optimization patterns

### 11. **Healthcare Patient Data Pipeline** (`healthcare-pipeline/`)
**Status**: ✅ **Available**  
**Industry**: Healthcare - Patient data processing and clinical decision support  
**Features**: HIPAA compliance, data validation, clinical workflows, emergency protocols  
**Real-world**: Epic Systems, Cerner healthcare data processing  
**Learn**: Compliance patterns, critical system design, healthcare workflows

## 🎯 Learning Path

### Beginner (Examples 1-3)
- Basic task execution and concurrent processing
- Error handling and timeout management
- Sequential workflow patterns

### Intermediate (Examples 4-7)
- Conditional orchestration and dynamic workflows
- Progress tracking and monitoring
- Resource management and optimization

### Advanced (Examples 8-11)
- Complex enterprise orchestration patterns
- Real-time processing and event-driven architectures
- Compliance and regulatory requirements

## 📊 Feature Matrix

| Example | Concurrent | Sequential | Conditional | Error Handling | Progress | Resource Mgmt | Industry Focus |
|---------|------------|------------|-------------|----------------|----------|---------------|----------------|
| 1. Hello | ✅ | ❌ | ❌ | Basic | Basic | ❌ | Learning |
| 2. Health Monitoring | ✅ | ✅ | ❌ | ⭐⭐⭐ | ✅ | ❌ | DevOps |
| 3. Order Processing | ❌ | ⭐⭐⭐ | ❌ | ⭐⭐⭐ | ✅ | ❌ | E-commerce |
| 4. Content Moderation | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ✅ | ✅ | ❌ | Social Media |
| 5. Financial Transactions | ✅ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ✅ | ❌ | FinTech |
| 6. Video Processing | ⭐⭐⭐ | ✅ | ❌ | ✅ | ⭐⭐⭐ | ⭐⭐⭐ | Media |
| 7. IoT Processing | ⭐⭐⭐ | ✅ | ✅ | ✅ | ⭐⭐⭐ | ⭐⭐⭐ | IoT |
| 8. CI/CD Pipeline | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ✅ | DevOps |
| 9. Fraud Detection | ⭐⭐⭐ | ✅ | ⭐⭐⭐ | ⭐⭐⭐ | ✅ | ⭐⭐⭐ | Banking |
| 10. Supply Chain | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | Logistics |
| 11. Healthcare | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | Healthcare |

**Legend**: ❌ Not used, ✅ Basic usage, ⭐⭐⭐ Advanced usage

## 🏭 Industry Applications

### DevOps & Infrastructure
- **Health Monitoring**: Service mesh monitoring and alerting
- **CI/CD Pipeline**: Deployment automation and testing

### E-commerce & Retail
- **Order Processing**: Order fulfillment and inventory management
- **Supply Chain**: Logistics optimization and procurement

### Financial Services
- **Transaction Processing**: Payment processing and compliance
- **Fraud Detection**: Real-time monitoring and risk assessment

### Media & Entertainment
- **Video Processing**: Content transcoding and distribution
- **Content Moderation**: Safety and compliance automation

### Healthcare & Life Sciences
- **Patient Data Pipeline**: Clinical data processing and decision support

### Manufacturing & IoT
- **IoT Processing**: Sensor data aggregation and analytics

### Social Media & Communication
- **Content Moderation**: Automated content safety and compliance

## 🚀 Quick Start Guide

1. **Clone the repository**
   ```bash
   git clone https://github.com/maniartech/orchestrator
   cd orchestrator/examples
   ```

2. **Start with Hello World**
   ```bash
   cd hello
   go run main.go
   ```

3. **Explore progressively**
   - Read each example's README.md
   - Run the code and observe the output
   - Modify parameters to see different behaviors
   - Apply patterns to your own use cases

## 📖 Documentation

- [Main Documentation](../README.md) - Core library documentation
- [API Reference](../docs/api.md) - Detailed API documentation
- [Best Practices](../docs/best-practices.md) - Production usage guidelines

## 🤝 Contributing

Found a bug or want to add an example? See our [Contributing Guide](../CONTRIBUTING.md).

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.

---

**Note**: All examples are now available and fully functional, demonstrating the comprehensive capabilities of the orchestrator library across various industry scenarios.
