package prototype

func CosToVesting(cos *Coin) *Vest{
	vest := &Vest{Amount:&Safe64{}}
	vest.Amount.Value = cos.Amount.Value
	return vest
}

func VestingToCoin(vest *Vest) *Coin {
	cos := &Coin{Amount:&Safe64{}}
	cos.Amount.Value = vest.Amount.Value
	return cos
}