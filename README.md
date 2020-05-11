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

* Go中有一个包叫unsafe,顾名思义，它不安全，因为它可以直接操作内存。我们可以使用unsafe，来对一个字符串进行赋值，具体的步骤大概如下：

** 获得该字符串的地址
** 对该地址赋值
  我们通过四行就可以完成上面的操作：
```
      str := ""
      // 获得该字符串的地址
      p := uintptr(unsafe.Pointer(&str))
      // 在该地址上赋值
      *(*string)(unsafe.Pointer(p))="test"
      fmt.Println(str)
  -----------------
  test
```
  当我们能够使用unsafe来操作内存时，就可以进一步尝试操作结构体了。

  操作结构体
  我们通过上述代码，得到一个结论：

-  只要我们知道内存地址，就可以操作任意变量。
   接下来我们可以尝试去操作结构体了。

  Go的结构体有以下的两个特点：

-  结构体的成员变量是顺序存储的
-  结构体第一个成员变量的地址就是该结构体的地址。
  根据以上两点，以及刚刚我们得到的结论，我们可能够得到以下的方法，来干掉value.Field().Set()

  获得结构体地址
  获得结构体内成员变量的偏移量
  得到结构体成员变量地址
  修改变量值
  我们逐个来获得获得。

  Go中interface类型是以这样的形式保存的：
```
  // emptyInterface is the header for an interface{} value.
  type emptyInterface struct {
      typ  *rtype
      word unsafe.Pointer
  }
```
  这个结构体的定义可以在reflect/Value.go找到。

  在这个结构体中typ是该interface的具体类型，word指针保存了指向结构体的地址。

  现在我们了解了interface的存储类型后，我们只需要将一个空接口interface{}转换为emptyInterface类型，然后得到其中的word，就可以拿到结构体的地址了，即解决了第一步。

  结构体类型强转
  先用下面这段代码示例，来解决一下不同结构体之间的转换：
```
  type Test1 struct {
      Test1 string
  }

  type Test2 struct {
      test2 string
  }

  func TestStruct(t *testing.T) {
      t1 := Test1{
          Test1: "hello",
      }

      t2 := *(*Test2)(unsafe.Pointer(&t1))
      fmt.Println(t2)
  }
  ----------------
  {hello}
```
  然后我们更换两个结构体中的成员变量类型，再尝试一下：
```
  type Test1 struct {
      a int32
      b []byte
  }

  type Test2 struct {
      b int16
      a string
  }

  func TestStruct(t *testing.T) {
      t1 := Test1{
          a:1,
          b:[]byte("asdasd"),
      }

      t2 := *(*Test2)(unsafe.Pointer(&t1))
      fmt.Println(t2)
  }
  ----------------
  {1 asdasd}
```
  我们可以发现，后面这次尝试两个结构体的类型完全不同，但是其中int32和int16的存储方式相同，[]byte和string的存储方式相同，我们可以得出一个简单的结论：

-  不论类型签名是否相同，只要底层存储方式相同，我们就可以强制转换，并且可以突破私有成员变量限制。
  通过上面我们得到的结论，可以将reflect/value.go里面的emptyInterface类型复制出来。然后我们对interface强转并取到word，就可以拿到结构体的地址了。
```
  type emptyInterface struct {
      typ  *struct{}
      word unsafe.Pointer
  }

  func TestStruct(t *testing.T) {
      var in interface{}
      in = People{
          Age:   18,
          Name:  "shiina",
          Test1: "test1",
          Test2: "test2",
      }

      t2 := uintptr(((*emptyInterface)(unsafe.Pointer(&in))).word)
      *(*int)(unsafe.Pointer(t2))=111
      fmt.Println(in)
  }
  ---------------
  {111 shiina test1 test2}
```
  我们获取了结构体地址后，根据结构体地址，修改了结构体内第一个成员变量的值，接下来我们开始进行第二步：得到结构体成员变量的偏移量

  我们可以通过反射，来轻松的获得每一个成员变量的偏移量，进而根据结构体的地址，获得每一个成员变量的地址。

  当我们获得了每一个成员变量的地址后，就可以很轻易的修改它了。
