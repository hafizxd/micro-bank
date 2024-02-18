# Micro Bank
An application with simple ideas about managing and transfer balance, with the purpose of learning all Go ecosystem and development flow correctly from start to end including Preventing DB Deadlock, Generating Query, Developing Web Server, Dockerizing, Implement CI/CD, Deploying to AWS, Containerization with Kubernetes, Expanse into Microservices with gRPC Protobuf, and Managing and Caching Jobs with Redis.

### Project Structure :
- api: code that handles web server including router, handler, middleware
- db: all interaction with database including migration, mocking, and sqlc query
- token: responsible for generating and validating access token
- util: utilities that would be used within application