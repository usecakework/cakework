# Python SDK for Cakework

## Local Development
```pip3 install .``` (in root of package, where pyproject.toml is located)


## Updating generated python protobufs
Generating the grpc files which we want to copy over to the src/cakework directory:

```
cd src/cakework
source env/bin/activate
pip install grpcio
pip install protobuf
pip install grpcio-tools
python3 -m grpc_tools.protoc -I. --python_out=. --pyi_out=. --grpc_python_out=. cakework.proto
```