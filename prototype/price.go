package prototype

func CosToVesting(cos *Coin) *Vest{
	vest := &Vest{}
	vest.Amount.Value = cos.Amount.Value
	return vest
}

func VestingToCoin(vest *Vest) *Coin {
	cos := &Coin{}
	cos.Amount.Value = vest.Amount.Value
	return cos
}