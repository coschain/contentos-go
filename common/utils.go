package common



func Int2Bytes(n uint32) []byte {
	var b []byte
	var i int
	for i=0;i<4;i++ {b=append(b,0)}
	i=4
	for (n>0 && i>0) {
		i--;
		b[i]=byte(n&0xff)
		n/=256
	}
	return b
}