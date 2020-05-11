* Go的反射性能极差，那么我们就来自己看一下它的性能和上一个我们正常创建Person对象比性能差了多少
```bazaar
go test -bench=.
goos: darwin
goarch: amd64
BenchmarkNew-12              	1000000000	         0.255 ns/op	       0 B/op	       0 allocs/op
BenchmarkNewUseReflect-12    	 8931004	       125 ns/op	      64 B/op	       2 allocs/op
PASS
ok  	_/Users/yg/workcode/my-github/go-interface-optimize/before	2.251s
```

* 我们猜测，反射性能的损耗具体分为两个部分，一个部分是reflect.New()，另一个部分是value.Field().Set(), 这时候我们可以使用Go原生自带的性能分析工具pprof来分析一下它们的主要耗时，来验证我们的猜测。我们对四个成员变量测试用例使用pprof

```
go test -bench=. -benchmem -memprofile memprofile.out -cpuprofile profile.out
goos: darwin
goarch: amd64
BenchmarkNew-12              	1000000000	         0.375 ns/op	       0 B/op	       0 allocs/op
BenchmarkNewUseReflect-12    	 7490292	       263 ns/op	      64 B/op	       2 allocs/op
PASS
ok  	_/Users/yg/workcode/my-github/go-interface-optimize/before	3.362s

go tool pprof ./profile.out
Type: cpu
Time: May 11, 2020 at 8:39pm (CST)
Duration: 2.74s, Total samples = 2.44s (89.09%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) list NewUseReflect
Total: 2.44s
ROUTINE ======================== _/Users/yg/workcode/my-github/go-interface-optimize/before.BenchmarkNewUseReflect in /Users/yg/workcode/my-github/go-interface-optimize/before/new_test.go
         0      1.57s (flat, cum) 64.34% of Total
         .          .     12:
         .          .     13:func BenchmarkNewUseReflect(b *testing.B) {
         .          .     14:	b.ReportAllocs()
         .          .     15:	b.ResetTimer()
         .          .     16:	for i := 0; i < b.N; i++ {
         .      1.57s     17:		NewUseReflect()
         .          .     18:	}
         .          .     19:}
ROUTINE ======================== _/Users/yg/workcode/my-github/go-interface-optimize/before.NewUseReflect in /Users/yg/workcode/my-github/go-interface-optimize/before/new.go
      90ms      1.57s (flat, cum) 64.34% of Total
         .          .     14:	}
         .          .     15:}
         .          .     16:
         .          .     17:func NewUseReflect() interface{} {
         .          .     18:	var p People
      20ms      420ms     19:	t := reflect.TypeOf(p)
         .      540ms     20:	v := reflect.New(t)
      40ms      270ms     21:	v.Elem().Field(0).Set(reflect.ValueOf(18))
      30ms      300ms     22:	v.Elem().Field(1).Set(reflect.ValueOf("shiina"))
         .       40ms     23:	return v.Interface()
         .          .     24:}
         .          .     25:
(pprof)
```

* 我们使用pprof得到了该函数的主要耗时，可以发现与我们的猜测无误，耗时主要分为三个部分：reflect.TypeOf(),reflect.New(),value.Field().Set(),其中我们可以把reflect.TypeOf()放到函数外，在初始化的时候生成，接下来我们主要关注value.Fidle().Set()


