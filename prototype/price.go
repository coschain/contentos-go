package prototype

func CosToVest(cos *Coin) *Vest {
	vest := &Vest{}
	vest.Value = cos.Value
	return vest
}

func VestToCoin(vest *Vest) *Coin {
	cos := &Coin{}
	cos.Value = vest.Value
	return cos
}
