package adapters

type DimensionAdapter struct {
	width, height int
}

func (d *DimensionAdapter) GetWidth() int {
	return d.width
}

func (d *DimensionAdapter) GetHeight() int {
	return d.height
}

func (d *DimensionAdapter) SetWidth(w int) {
	d.width = w
}

func (d *DimensionAdapter) SetHeight(h int) {
	d.height = h
}

func (d *DimensionAdapter) SetSize(width, height int) {
	d.SetWidth(width)
	d.SetHeight(height)
}
