package prototype

func (this *AccountName) Empty() bool {
	return 0 == this.Value.Lo && 0 == this.Value.Hi
}
