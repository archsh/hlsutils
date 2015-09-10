package stack

type Stack struct {
	idx  int
	data *[]int
}

func New(n int) (s *Stack){
	s = new Stack
	s.data = make([]int, n)
	return
}

func (s *Stack) Push(n int) {
	s.data[s.idx] = n
	s.idx++

}

func (s *Stack) Pop() (ret int) {
	s.idx--
	ret = s.data[s.idx]
	return
}
