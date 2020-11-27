
# Levity remote task runner  

## High-Level Description

`Levity` is a (small part of a) distributed task-running system, consisting of:

1. a server application that can schedule and monitor the execution of arbitrary processes, & an API that exposes that functionality, and
2. a client application that uses the API to interact with the service.

## Intended Use

`Levity` is intended to be used as an exercise as part of the Gravitational interview process. As this is an exercise, most design tradeoffs will be made in favour of simple implementation over producing an infinitely-robust, battle-hardened production system.

That said, the code that *is* implemented is expected to be robust, clean, modular, and race-free.

## Implementation Language

Go 1.15

## Repo Layout 

The project will use the [Within Go Repo Layout](https://christine.website/blog/within-go-repo-layout-2020-09-07).

## RPC mechanism

`Levity` will use gRPC as its underlying RPC mechanism. The reasons for this are:

* gRPC has built-in support for secure transport and authentication mechanisms, obviating the need to implement these features independently, or wrap them manually around a web service,
* gRPC mandates the use of a Protocol Buffer description of the service, and having an implementation-independent source of truth about the API is always useful, and
* I've never used it before, so it seems as good a time to learn as any.

## API

The API exposed by the server is described by the protocol buffer specification [here](../api/levity.proto).

## Task Lifecycle

1. Task is started by a call to `StartTask`.
2. The user can monior the task execution by repeatedly having the client
   poll the server via `QueryTask`.
3. The client may elect to kill task at any time with `SignalTask`. The
   server will try to shut it down gracefully at first and then brutally
   after some (server-specified) timeout.
4. The client may fetch the logs with `FetchLogs` at any time after
   starting the task, but will always receive the entire log at the time
   of the call. There is no mechanism to retreive a partial log.
5. After some inactivity timeout, the task record on the server is deleted.
   This implies that the task, if still running, is killed and the logs are no longer retrievable.

## Security Concerns

### Transport

The system will use TLS 1.3 for transport level security. The server will
require a certificate issued by a CA trusted by any potential clients.

Providing a system for issuing a certificate and having users install the CA root certificate is beyond the scope of this exercise.
### Authentication

Users will authenticate using mutual TLS. This mechanism should be provided out-of-the-box by the Go gRPC implementation. 

Mutual TLS requires that both the server and users require a certificate from a mutually-trusted CA in order to gain access to the API.

Systems for user certificate issuance and revocation is considered beyond the scope of this exercise.

### Permissions Model

This system will have a very simple permissions model, at least in the first instance.

* Any user deemed legitimate will be allowed to start a task
* Only the user that started a task may interact with it, (e.g. query status)

The obvious extension to this model is some form of Admin role that can query other users' tasks, but that is not being considered as part of this work.

### System Integrity & Availability

This system is assumed to be used by trusted users, and no effort will be made to prevent users damaging the system that is running these tasks. This includes actions like deleting resources, DoSing the system with a fork bomb, or any other harmful activity.

Similarly, no effort will be made to prevent the user exfiltrating data from the server (e.g. running `cat ~/.ssh/id_rsa`).

The above risks can be somewhat mitigated by restricting the permissions of the user running the API server.

### Testing Considerations

This system will use a self-generated CA root certificate for testing purposes. 

Ideally, both the client and server will have a configuration option to trust an arbitrary CA in order to run automated tests. Note that this specified CA *may* be hardcoded into the client and server binaries, depending on the time available for implementation. This is obviously something that would not be allowed in a production system.

Time permitting, the project makefile will provide a target to generate both the CD root certificates and derived client & server certificates from scratch.

## Implementation Details

### Server

#### Process Execution
Process execution will be managed via the Go `os.Process` type, most likely via the `os/exec` package. 

No special effort will be made to ensure that all tasks are cleanly shut down if the server crashes. 

#### Task Registration 
There will be a single, central in-memory task registry, protected by a simple reader-writer lock. The assumption is that the values will be looked up more often created or destroyed, so the reader-writer should reduce contention on the registry.

This has the value of simplicity, but it does mean that all task information is lost when the server is killed or crashes. It is easy to imagine this simple, in-memory mechanism being replaced by a database of some kind for increased durability.

#### Task Handle generation

For the purposes of this exercise, task IDs will be generated as UUIDs, with a
collision detection & re-generation mechanism to handle the remote chance of a UUID collision.

#### Individual Task Data 
The tasks themselves will have their own, individual locking mechanism, so that concurrent requests on different tasks should not contend for the same lock (once the tasks are retrieved from the registry, that is).

At present I am planning on using a reader/writer lock for each task, but this may change during implementation, depending on the complexity required and how much contention falls on each task.

Logs from `stdout` and `stderr` will be treated as opaque binary data. The logs will also be stored in memory only. This again favours ease of implementation over durability, but it is easy to imagine a system where the logs are written to files or a database, and read back from there on request.

#### Configuration

Any configuration options, including any sort of user database, it will be loaded once on startup (either via command line options or (potentially) via a config file), and no configuration modifications of any kind will be noticed during runtime.

### Client
#### UX
The MVP user interface is planned to be a simple CLI, with sub-commands for starting, querying and stopping a task, for example:
```
$ levity start cat /root/.ssh/id_rsa
[some-task-handle]
$ levity status [some-task-handle]
exited
stdout:
    Access Denied
```
#### Configuration
The client will use a CLI to receive all configuration data (with the exception of a possible user database, if necessary) for all interactions with the server. Extensions could be made for a complete configuration system that merges CLI, config files and environment variables into a single configuration (e.g via something like [Viper](https://github.com/spf13/viper)), but that is not planned for now.

A CLI is easy to write, and persistent configuration can be layered on top of it with shell scripts if necessary.
