from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class CallTaskReply(_message.Message):
    __slots__ = ["error", "requestId"]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    REQUESTID_FIELD_NUMBER: _ClassVar[int]
    error: str
    requestId: str
    def __init__(self, requestId: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class CallTaskRequest(_message.Message):
    __slots__ = ["app", "parameters", "task", "userId"]
    APP_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    TASK_FIELD_NUMBER: _ClassVar[int]
    USERID_FIELD_NUMBER: _ClassVar[int]
    app: str
    parameters: str
    task: str
    userId: str
    def __init__(self, userId: _Optional[str] = ..., app: _Optional[str] = ..., task: _Optional[str] = ..., parameters: _Optional[str] = ...) -> None: ...
