# levity

## Building the Client and Server

```
$ make binaries
```

The server (`levityd`) and client (`levity`) binaries will be built into
the `bin` directory. All of the exampes below assume that both of these
binaries are available on your search path.

## Running the Server

**WARNING** Running this service will allow people to execute *arbitrary 
commands* over the network on your computer. Anything you can do, an
attacker can do (e.g.: `rm -rf /`). **Take appropriate precautions.**

Start the server by running `levityd`, passing the address to bind to as
the first and only argument, like so:

```
$ levityd 0.0.0.0:4321
```

## Using the Client

Note: The commands shown below are all descriptive examples, and will
not be what you want to run in practice. Go get the definitive documentation
on any command, consult `levity --help`, or the `help` command, e.g. 
`levity help start`.

In all cases, `levity` will exit with a `0` exit code on success, or a
nonzero exit code on failure.

### Starting a task

Using the `start` command will start a task on the server, returning a task
ID that can be used to interact with the task later.

```
$ levity -a example.com:4321 start ls /home/trent
f257cd86-8ec6-4688-b902-2a118e0a3035
```

*Tip:* If you want to pass flags to the task, you can use the standard `--`
end-of-flags marker to bypass the `levity` flag parser and give flags arguments 
to the task on the server, e.g.

```
$ levity -a example.com:4321 start -- ls -la /home/trent
22163af1-e04f-468b-88a5-c4211007cb67
```

See `levity help start` for more information
### Querying a task state
To query the state of the task use the `query` command:

```
$ levity - query [task-id]
Finished
0
```

The first line of the `query` output is the task state, one of :
 * `NotStarted`: The task has been created, but the underlying process is not yet started.
 * `Running`: The task is running normally
 * `Signalled`: Quit requested, but not yet exited
 * `Finished`: Task exited naturally, either of its own volition or after being signalled. 
 * `BrutallyKilled`: The task refused to respond to a signal, and has now been killed. 
 * `InternalServerError` the tsk failed tue to an unexpected error in the server.

The second line, if present, shows the task exit code. This will always be an integer.

### Stopping a task.

To stop a long-running task use the `signal` command:

```
$ levity -a example.com:4321 signal f257cd86-8ec6-4688-b902-2a118e0a3035
```

This will issue a soft request for the task to quit by signalling the underlying 
process with a SIGTERM. The server ives the task a 5-second grace period to clean
up any resources it might have and exit. If the process has _not_ exited after 5
seconds, the server will kill it with a SIGKILL.

Note that `signal` _does not wait_ for the task to exit. You will need to monitor
it with `query` to detect when it exits.

### Fetching task output

Fetch task output with the `logs` command, e.g.

```
$ levity -a example.com:4321 logs f257cd86-8ec6-4688-b902-2a118e0a3035
drwxr-xr-x  20 trent  staff      680 Dec 21 23:01 .
drwxr-xr-x  75 trent  staff     2550 Dec 15 00:29 ..
drwxr-xr-x  15 trent  staff      510 Dec 22 23:38 .git
...
```

The `logs` command fetches both the stdout and stderr streams, and writes 
them to the same streams on the client, i.e. data from the task's stdout 
output is written to the `levity` client's stdout, and the task `stderr`
likewise goes to the local stderr.

## Running tests

The unit tests for the `task` package require that some
specific binaries exist on the search path. The easiest way
to make sure that this is set up correctly is to run the tests form the makefile:

```
$ make tests
```

You can also run the unit tests with the race checker enabled by `make`ing the `tests-race` target:

```
$ make tests-race
```