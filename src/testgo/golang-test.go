package main

import "fmt"
import "net/url"
import "unicode/utf8"
import "hls/m3u8"


func main() {
	const A = 100
	u1, _ := url.Parse("http://stb-video.tv-cloud.cn:7070/tvseeHandle/auth.go")
	u2, _ := u1.Parse("http://www.tvsee.cn/test.ts")
	fmt.Printf("A = %v\n", A)
	fmt.Printf("u1 = %v\n", u1)
	fmt.Printf("u2 = %v\n", u2)

	list := []string{ "a", "b", "c", "d", "e", "f" }
	for k, v := range list {
		fmt.Printf("list[%v] = %v \n", k, v)
	}

	for p, c := range "This is a test!" {
		fmt.Printf("%v: %c \n", p, c)
	}

	arr := []int{0,1,2,3,4,5,6,7,8,9}
	s1 := arr[2:4]
	s2 := arr[5:8]

	fmt.Printf("arr: len=%d, cap=%d \n", len(arr), cap(arr))
	fmt.Printf("s1: len=%d, cap=%d \n", len(s1), cap(s1))
	fmt.Printf("s2: len=%d, cap=%d \n", len(s2), cap(s2))

	for i := 0; i<10; i++ {
		fmt.Printf("%d\t", i)
	}
	fmt.Printf("\n")

	k := 0
	GOGOGO:
	fmt.Printf("%d\t", k)
	k++
	if k<10 {
		goto GOGOGO
	}
	fmt.Printf("\n")
	for i:=1; i<=100; i++ {
		switch {
		case i%3==0 && i%5==0:
			fmt.Printf("FizzBuzz")
		case i%3==0:
			fmt.Printf("Fizz")
		case i%5==0:
			fmt.Printf("Buzz")
		default:
			fmt.Printf("%d", i)
		}
		fmt.Printf("\n")
	}

	// for i:=0;i<100;i++{
	// 	for j:=0;j<=i;j++{
	// 		fmt.Printf("A")
	// 	}
	// 	fmt.Printf("\n")
	// }

	s := "asSASA ddd dsjkdsjs dk"
	fmt.Printf("length of s = %d \n", len(s))
	sl := []rune(s)
	n := 0
	for _,c := range sl {
		if c != ' '{
			n++
		}
	}
	fmt.Printf("num of chars = %d : %d \n", n, utf8.RuneCountInString(s))
	sl[4], sl[5], sl[6] = 'a', 'b', 'c'
	fmt.Printf("Old string: %s \n", s)
	fmt.Printf("New string: %s \n", string(sl))

	ss := "foobar"
	ssl := []rune(ss)
	for i:=0;i<len(ss)/2;i++{
		ssl[i], ssl[len(ss)-1-i] = ssl[len(ss)-1-i], ssl[i]
	}
	fmt.Printf("New foobar: %s \n", string(ssl))

	flts := []float64{1012.12, 2341.234, 3412.5, 141234.42}
	var avg float64
	for _, v := range flts {
		avg += v
	}
	fmt.Printf("Avg of floats: %f \n", avg/float64(len(flts)))

	defer_test()

	x1 := throwsPanic(func(){
		return
		})
	x2 := throwsPanic(func(){
		panic("Great")
		})
	fmt.Printf("x1=%v, x2=%v \n", x1, x2)

	fmt.Printf("Avg of 1.0, 2.0, 3.0 = %f \n", theavg(1,2,3))
	var x, y int
	x, y = ordered(2,5) 
	fmt.Printf("Ordered(2,5) = %d, %d \n", x,y)
	x, y = ordered(4,3) 
	fmt.Printf("Ordered(4,3) = %d, %d \n", x,y)
	x, y = ordered(6,6) 
	fmt.Printf("Ordered(6,6) = %d, %d \n", x,y)

	fmt.Printf("Fibonacci(20)=%v\n", Fibonacci(20))
	fmt.Printf("Maped(1,2,3,4,5,6,7)=%v\n", Map(func (n int) int {
			return n*2
		}, []int{1,2,3,4,5,6,7}))
	fmt.Printf("Max(1,2,3,4,5,6,7,8,9)=%d\n", Max([]int{1,2,3,4,5,6,7,8,9}))
	fmt.Printf("Min(1,2,3,4,5,6,7,8,9)=%d\n", Min([]int{1,2,3,4,5,6,7,8,9}))
	fmt.Printf("BubbleSort(3,4,6,7,8,21,452,745,8432,1,2,53)=%v\n", BubbleSort([]int{3,4,6,7,8,21,452,745,8432,1,2,53}))
	m3u8.Decode("This is just a test!")
}


func defer_test() {
	for i:=0; i<10; i++{
		defer fmt.Printf("%d\t", i)
	}
}

func throwsPanic(f func()) (b bool) {
	defer func(){
		if x:= recover(); x != nil {
			b = true
		}
	}()
	f()
	return
}

func theavg(arg...float64) (result float64){
	var sum float64
	for _, v := range arg {
		sum += v
	}
	result = sum/float64(len(arg))
	return
}

func ordered(x, y int) (m , n int){
	if x > y {
		m, n = y, x
	}else if x < y {
		m, n = x, y
	}else{
		m, n = x, y
	}
	return
}

func Fibonacci(n int) ([]int) {
	if n > 2 {
		s := Fibonacci(n-1)
		return append(s, s[len(s)-1]+s[len(s)-2])
	}else if n==2{
		return []int{1,1}
	}else if n==1{
		return []int{1}
	}else{
		return []int{}
	}
}

func Map(f func (int)(int), s []int) []int {
	result := make([]int, len(s))
	for i, v := range s {
		result[i] = f(v)
	}
	return result
}

func Min(s []int) (m int) {
	m = s[0]
	for _,v := range s {
		if v < m {
			m = v
		}
	}
	return
}

func Max(s []int) (m int) {
	m = s[0]
	for _,v := range s {
		if v > m {
			m = v
		}
	}
	return
}

func BubbleSort(s []int) []int{
	for i:=0;i<len(s);i++{
		for j:=i+1; j<len(s);j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
	return s
}