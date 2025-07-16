package payment

import "errors"


var ErrAllProcessorsAreDown = errors.New("all payment processors are down; try again later")
