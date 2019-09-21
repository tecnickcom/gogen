// Package ~#PROJECT#~
// Copyright (c) ~#CURRENTYEAR#~ ~#OWNER#~
// ~#SHORTDESCRIPTION#~
package ~#LIBPACKAGE#~

// Info contains ...
type Info struct {
	Desc string `json:"desc"` // description
}

// GetDesc returns the Desc field
func (i *Info) GetDesc() string {
	return i.Desc
}
