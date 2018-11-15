package prototype

func CosToVesting(cos *Coin) *Vest{
	vest := &Vest{}
	vest.Value = cos.Value
	return vest
}

func VestingToCoin(vest *Vest) *Coin {
	cos := &Coin{}
	cos.Value = vest.Value
	return cos
}