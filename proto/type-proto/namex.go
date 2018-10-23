package prototype

func (this *Namex) Empty() bool {
	return 0 == this.Value.Lo && 0 == this.Value.Hi
}
