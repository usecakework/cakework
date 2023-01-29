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
    __slots__ = ["parameters", "project", "runId", "userId"]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    RUNID_FIELD_NUMBER: _ClassVar[int]
    USERID_FIELD_NUMBER: _ClassVar[int]
    parameters: str
    project: str
    runId: str
    userId: str
    def __init__(self, parameters: _Optional[str] = ..., userId: _Optional[str] = ..., project: _Optional[str] = ..., runId: _Optional[str] = ...) -> None: ...
