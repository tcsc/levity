// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package api

import (
	context "context"
	empty "github.com/golang/protobuf/ptypes/empty"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// TaskManagerClient is the client API for TaskManager service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type TaskManagerClient interface {
	// StartTask attempts to launch a task on the server, returning a handle
	// to interact with the task via the other API end points. he launched
	// process will run under, and inherit the permissions of, the account
	// of the user running the API server. No shell variable substitution
	// will be performed.
	StartTask(ctx context.Context, in *StartTaskRequest, opts ...grpc.CallOption) (*StartTaskResponse, error)
	// QueryTask fetches the current state of the task, e.g. Running,
	// Finished, etc.
	QueryTask(ctx context.Context, in *QueryTaskRequest, opts ...grpc.CallOption) (*QueryTaskResponse, error)
	// SignalTasks requests that the task exit. The process will be given a
	// SIGTERM, and if the process does not exit after an arbitrary timeout it
	// will be killed. Note that this call will NOT wait on the task to
	// finish; the caller will need to poll with QueryTask to determine when
	// the task has finished.
	SignalTask(ctx context.Context, in *SignalTaskRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	// FetchLogs returns the data written to stdout and stderr by the task. The
	// log data from each stream is treated as an opaque series of bytes
	FetchLogs(ctx context.Context, in *FetchLogsRequest, opts ...grpc.CallOption) (*FetchLogsResponse, error)
}

type taskManagerClient struct {
	cc grpc.ClientConnInterface
}

func NewTaskManagerClient(cc grpc.ClientConnInterface) TaskManagerClient {
	return &taskManagerClient{cc}
}

func (c *taskManagerClient) StartTask(ctx context.Context, in *StartTaskRequest, opts ...grpc.CallOption) (*StartTaskResponse, error) {
	out := new(StartTaskResponse)
	err := c.cc.Invoke(ctx, "/levity.TaskManager/StartTask", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskManagerClient) QueryTask(ctx context.Context, in *QueryTaskRequest, opts ...grpc.CallOption) (*QueryTaskResponse, error) {
	out := new(QueryTaskResponse)
	err := c.cc.Invoke(ctx, "/levity.TaskManager/QueryTask", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskManagerClient) SignalTask(ctx context.Context, in *SignalTaskRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/levity.TaskManager/SignalTask", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskManagerClient) FetchLogs(ctx context.Context, in *FetchLogsRequest, opts ...grpc.CallOption) (*FetchLogsResponse, error) {
	out := new(FetchLogsResponse)
	err := c.cc.Invoke(ctx, "/levity.TaskManager/FetchLogs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// TaskManagerServer is the server API for TaskManager service.
// All implementations must embed UnimplementedTaskManagerServer
// for forward compatibility
type TaskManagerServer interface {
	// StartTask attempts to launch a task on the server, returning a handle
	// to interact with the task via the other API end points. he launched
	// process will run under, and inherit the permissions of, the account
	// of the user running the API server. No shell variable substitution
	// will be performed.
	StartTask(context.Context, *StartTaskRequest) (*StartTaskResponse, error)
	// QueryTask fetches the current state of the task, e.g. Running,
	// Finished, etc.
	QueryTask(context.Context, *QueryTaskRequest) (*QueryTaskResponse, error)
	// SignalTasks requests that the task exit. The process will be given a
	// SIGTERM, and if the process does not exit after an arbitrary timeout it
	// will be killed. Note that this call will NOT wait on the task to
	// finish; the caller will need to poll with QueryTask to determine when
	// the task has finished.
	SignalTask(context.Context, *SignalTaskRequest) (*empty.Empty, error)
	// FetchLogs returns the data written to stdout and stderr by the task. The
	// log data from each stream is treated as an opaque series of bytes
	FetchLogs(context.Context, *FetchLogsRequest) (*FetchLogsResponse, error)
	mustEmbedUnimplementedTaskManagerServer()
}

// UnimplementedTaskManagerServer must be embedded to have forward compatible implementations.
type UnimplementedTaskManagerServer struct {
}

func (UnimplementedTaskManagerServer) StartTask(context.Context, *StartTaskRequest) (*StartTaskResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StartTask not implemented")
}
func (UnimplementedTaskManagerServer) QueryTask(context.Context, *QueryTaskRequest) (*QueryTaskResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method QueryTask not implemented")
}
func (UnimplementedTaskManagerServer) SignalTask(context.Context, *SignalTaskRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SignalTask not implemented")
}
func (UnimplementedTaskManagerServer) FetchLogs(context.Context, *FetchLogsRequest) (*FetchLogsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FetchLogs not implemented")
}
func (UnimplementedTaskManagerServer) mustEmbedUnimplementedTaskManagerServer() {}

// UnsafeTaskManagerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to TaskManagerServer will
// result in compilation errors.
type UnsafeTaskManagerServer interface {
	mustEmbedUnimplementedTaskManagerServer()
}

func RegisterTaskManagerServer(s grpc.ServiceRegistrar, srv TaskManagerServer) {
	s.RegisterService(&_TaskManager_serviceDesc, srv)
}

func _TaskManager_StartTask_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartTaskRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TaskManagerServer).StartTask(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/levity.TaskManager/StartTask",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TaskManagerServer).StartTask(ctx, req.(*StartTaskRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _TaskManager_QueryTask_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryTaskRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TaskManagerServer).QueryTask(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/levity.TaskManager/QueryTask",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TaskManagerServer).QueryTask(ctx, req.(*QueryTaskRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _TaskManager_SignalTask_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignalTaskRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TaskManagerServer).SignalTask(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/levity.TaskManager/SignalTask",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TaskManagerServer).SignalTask(ctx, req.(*SignalTaskRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _TaskManager_FetchLogs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FetchLogsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TaskManagerServer).FetchLogs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/levity.TaskManager/FetchLogs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TaskManagerServer).FetchLogs(ctx, req.(*FetchLogsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _TaskManager_serviceDesc = grpc.ServiceDesc{
	ServiceName: "levity.TaskManager",
	HandlerType: (*TaskManagerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "StartTask",
			Handler:    _TaskManager_StartTask_Handler,
		},
		{
			MethodName: "QueryTask",
			Handler:    _TaskManager_QueryTask_Handler,
		},
		{
			MethodName: "SignalTask",
			Handler:    _TaskManager_SignalTask_Handler,
		},
		{
			MethodName: "FetchLogs",
			Handler:    _TaskManager_FetchLogs_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/levity.proto",
}
