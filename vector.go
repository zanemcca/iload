package main

/*
  Define a vector type that has a contains method
*/
var zeroStruct struct{}

type Vector map[string]struct{}

func (v Vector) add(key string) {
	v[key] = zeroStruct
}

func (v Vector) contains(key string) bool {
	val, ok := v[key]
	_ = val
	return ok
}

func newVector(keys ...string) Vector {
	v := make(Vector)
	for i := 0; i < len(keys); i++ {
		v.add(keys[i])
	}
	return v
}
