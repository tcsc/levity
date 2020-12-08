# levity

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