package midi

type MyRange struct {
	cnt int

	lowerBound int64
	upperBound int64
}

func newMyRange(lowerBound int64, upperBound int64) *MyRange {
	return &MyRange{
		lowerBound: lowerBound,
		upperBound: upperBound,
	}
}

func (m *MyRange) stepBy(n int) {
	m.cnt += n
	step := m.upperBound - m.lowerBound

	m.upperBound += step * int64(n)
	m.lowerBound += step * int64(n)
}

func (m *MyRange) contains(item int64) bool {
	return item >= m.lowerBound && item < m.upperBound
}

func (m *MyRange) position() int {
	return m.cnt % 4
}
