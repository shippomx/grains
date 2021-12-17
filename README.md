# grains
Goroutine stacks analysis tool, trim a large goroutine stack file, and generated it's summary.
If dead locks pair exists, warning us the surspicous goroutines.

command `trim` generate the summary.
command `show [goutine_id]` print the goroutine details.

## 
```bash
$ grains dockerd.log

Entering interactive mode (type "help" for commands, "o" for options)
(grains) trim
================= Summary =================
blocked goroutine types:
runnable: 42
running: 1
semacquire: 330
================= WARNING DEAD LOCK semacquire =================
goroutine 66926 has surspicous DEAD LOCK with goroutine 67777
LockHolders of goroutine 66926: [*State *Daemon *memoryStore]
LockHolders of goroutine 67777: [*memoryStore *Daemon]
chan receive: 64
syscall: 330
select: 172
sleep: 2
IO wait: 510
(grains) show 66926
================= goroutine 66926 =================
goroutine 66926 [semacquire, 2031 minutes]:
sync.runtime_SemacquireMutex(0xc0002eae04, 0xc003b53100, 0x1)
t/usr/local/go/src/runtime/sema.go:71 +0x47
sync.(*Mutex).lockSlow(0xc0002eae00)
t/usr/local/go/src/sync/mutex.go:138 +0x105
sync.(*Mutex).Lock(...)()
t       /usr/local/go/src/sync/mutex.go:81
github.com/docker/docker/container.(*State).IsRunning(0xc0002eae00, 0x1673180)
t/root/rpmbuild/BUILD/docker-ce/.gopath/src/github.com/docker/docker/container/state.go:238 +0x77
github.com/docker/docker/daemon.(*Daemon).ImageDelete.func1(0xc00348ce00, 0xc0008dcd00)
t/root/rpmbuild/BUILD/docker-ce/.gopath/src/github.com/docker/docker/daemon/image_delete.go:87 +0x45
github.com/docker/docker/container.(*memoryStore).FilterAll(0xc00011fd60, 0xc0037d9140, 0x47, 0x3, 0x0)
t/root/rpmbuild/BUILD/docker-ce/.gopath/src/github.com/docker/docker/container/memory_store.go:74 +0x147
github.com/docker/docker/daemon.(*Daemon).ImageDelete(0xc00018c480, 0xc000c421d5, 0xc, 0x10100, 0xc00089e3f8, 0x7f66303bd1d0, 0x8, 0x10, 0x7f66636022f0)
t/root/rpmbuild/BUILD/docker-ce/.gopath/src/github.com/docker/docker/daemon/image_delete.go:90 +0x2
(grains)  
```
