package prototype

import (
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestVest(t *testing.T){

	a := assert.New(t)

	v1 := NewVest( math.MaxUint64 - 10)
	v2 := NewVest( 20 )

	v1.Sub(v2)
	v1.Add(v2)

	func(){
		defer func() {
			err := recover()
			a.True( err != nil )
		}()
		v1.Add(v2)
	}()

	func(){
		defer func() {
			err := recover()
			a.True( err != nil )
		}()
		v1.Mul(2)
	}()


	func(){
		defer func() {
			err := recover()
			a.True( err != nil )
		}()
		v2.Sub(v1)
	}()

}

func TestVestZero(t *testing.T){
	a := assert.New(t)
	v := NewVest( 0 )

	v.Mul(100)
	a.True( v.Value == 0 )
}

func TestCoinZero(t *testing.T){
	a := assert.New(t)
	v := NewCoin( 0 )

	v.Mul(100)
	a.True( v.Value == 0 )
}


func TestCoin(t *testing.T){

	a := assert.New(t)

	v1 := NewCoin( math.MaxUint64 - 10)
	v2 := NewCoin( 20 )

	v1.Sub(v2)
	v1.Add(v2)

	func(){
		defer func() {
			err := recover()
			a.True( err != nil )
		}()
		v1.Add(v2)
	}()

	func(){
		defer func() {
			err := recover()
			a.True( err != nil )
		}()
		v1.Mul(2)
	}()

	func(){
		defer func() {
			err := recover()
			a.True( err != nil )
		}()
		v2.Sub(v1)
	}()
}