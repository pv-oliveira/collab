package models

import "errors"

// Erros de domínio: o repository traduz erros do banco para estes, e o
// handler traduz estes para status HTTP. Nenhuma camada conhece a outra.
var ErrEmailTaken = errors.New("email already registered")
