from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Reply(_message.Message):
    __slots__ = ["result"]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: str
    def __init__(self, result: _Optional[str] = ...) -> None: ...

class Request(_message.Message):
    __slots__ = ["app", "parameters", "requestId", "userId"]
    APP_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    REQUESTID_FIELD_NUMBER: _ClassVar[int]
    USERID_FIELD_NUMBER: _ClassVar[int]
    app: str
    parameters: str
    requestId: str
    userId: str
    def __init__(self, parameters: _Optional[str] = ..., userId: _Optional[str] = ..., app: _Optional[str] = ..., requestId: _Optional[str] = ...) -> None: ...
