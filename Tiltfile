# Load the restart_process extension
load('ext://restart_process', 'docker_build_with_restart')

### K8s Config ###

# Uncomment to use secrets
# k8s_yaml('./infra/development/k8s/secrets.yaml')

k8s_yaml('./infra/development/k8s/app-config.yaml')

### End of K8s Config ###
### API Gateway ###

gateway_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/api-gateway ./services/api-gateway'
if os.name == 'nt':
  gateway_compile_cmd = './infra/development/docker/api-gateway-build.bat'

local_resource(
  'api-gateway-compile',
  gateway_compile_cmd,
  deps=['./services/api-gateway', './shared'], labels="compiles")


docker_build_with_restart(
  'askshop/api-gateway',
  '.',
  entrypoint=['/app/build/api-gateway'],
  dockerfile='./infra/development/docker/api-gateway.Dockerfile',
  only=[
    './build/api-gateway',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/api-gateway-deployment.yaml')
k8s_resource('api-gateway', port_forwards=8081,
             resource_deps=['api-gateway-compile'], labels="services")
### End of API Gateway ###
### Product Service ###
product_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/product-service ./services/product-service/cmd/main.go'
if os.name == 'nt':
  product_compile_cmd = './infra/development/docker/product-build.bat'
local_resource(
  'product-service-compile',
  product_compile_cmd,
  deps=['./services/product-service', './shared'], labels="compiles")
docker_build_with_restart(
  'askshop/product-service',
  '.',
  entrypoint=['/app/build/product-service'],
  dockerfile='./infra/development/docker/product-service.Dockerfile',
  only=[
    './build/product-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)
k8s_yaml('./infra/development/k8s/product-service-deployment.yaml')
k8s_resource('product-service', port_forwards=[8082, 9090],
             resource_deps=['product-service-compile'], labels="services")

### End of Product Service ###
### Cart Service ###
cart_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/cart-service ./services/cart-service/cmd/main.go'
if os.name == 'nt':
  cart_compile_cmd = './infra/development/docker/cart-build.bat'
local_resource(
  'cart-service-compile',
  cart_compile_cmd,
  deps=['./services/cart-service', './shared'], labels="compiles")
docker_build_with_restart(
  'askshop/cart-service',
  '.',
  entrypoint=['/app/build/cart-service'],
  dockerfile='./infra/development/docker/cart-service.Dockerfile',
  only=[
    './build/cart-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)
k8s_yaml('./infra/development/k8s/cart-service-deployment.yaml')
k8s_resource('cart-service', port_forwards=8083,
              resource_deps=['cart-service-compile'], labels="services")
### End of Cart Service ###
### AI Service ###
ai_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/ai-service ./services/ai-service/cmd/main.go'
if os.name == 'nt':
  ai_compile_cmd = './infra/development/docker/ai-build.bat'
local_resource(
  'ai-service-compile',
  ai_compile_cmd,
  deps=['./services/ai-service', './shared'], labels="compiles")
docker_build_with_restart(
  'askshop/ai-service',
  '.',
  entrypoint=['/app/build/ai-service'],
  dockerfile='./infra/development/docker/ai-service.Dockerfile',
  only=[
    './build/ai-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)
k8s_yaml('./infra/development/k8s/ai-service-deployment.yaml')
k8s_resource('ai-service', port_forwards=8086,
              resource_deps=['ai-service-compile'], labels="services")
### End of AI Service ###
### User Service ###
user_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/user-service ./services/user-service/cmd/main.go'
if os.name == 'nt':
  user_compile_cmd = './infra/development/docker/user-build.bat'
local_resource(
  'user-service-compile',
  user_compile_cmd,
  deps=['./services/user-service', './shared'], labels="compiles")
docker_build_with_restart(
  'askshop/user-service',
  '.',
  entrypoint=['/app/build/user-service'],
  dockerfile='./infra/development/docker/user-service.Dockerfile',
  only=[
    './build/user-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)
k8s_yaml('./infra/development/k8s/user-service-deployment.yaml')
k8s_resource('user-service', port_forwards=8084,
              resource_deps=['user-service-compile'], labels="services")
### End of User Service ###
### Order Service ###
order_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/order-service ./services/order-service/cmd/main.go'
if os.name == 'nt':
  order_compile_cmd = './infra/development/docker/order-build.bat'
local_resource(
  'order-service-compile',
  order_compile_cmd,
  deps=['./services/order-service', './shared'], labels="compiles")
docker_build_with_restart(
  'askshop/order-service',
  '.',
  entrypoint=['/app/build/order-service'],
  dockerfile='./infra/development/docker/order-service.Dockerfile',
  only=[
    './build/order-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)
k8s_yaml('./infra/development/k8s/order-service-deployment.yaml')
k8s_resource('order-service', port_forwards=8085,
              resource_deps=['order-service-compile'], labels="services")
### End of Order Service ###


### Trip Service ###

# Uncomment once we have a trip service

#trip_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/trip-service ./services/trip-service/cmd/main.go'
#if os.name == 'nt':
#  trip_compile_cmd = './infra/development/docker/trip-build.bat'

# local_resource(
#   'trip-service-compile',
#   trip_compile_cmd,
#   deps=['./services/trip-service', './shared'], labels="compiles")

# docker_build_with_restart(
#   'ride-sharing/trip-service',
#   '.',
#   entrypoint=['/app/build/trip-service'],
#   dockerfile='./infra/development/docker/trip-service.Dockerfile',
#   only=[
#     './build/trip-service',
#     './shared',
#   ],
#   live_update=[
#     sync('./build', '/app/build'),
#     sync('./shared', '/app/shared'),
#   ],
# )

# k8s_yaml('./infra/development/k8s/trip-service-deployment.yaml')
# k8s_resource('trip-service', resource_deps=['trip-service-compile'], labels="services")

### End of Trip Service ###
### Web Frontend ###

# docker_build(
#   'ride-sharing/web',
#   '.',
#   dockerfile='./infra/development/docker/web.Dockerfile',
# )

# k8s_yaml('./infra/development/k8s/web-deployment.yaml')
# k8s_resource('web', port_forwards='3001:3000', labels="frontend")

### End of Web Frontend ###