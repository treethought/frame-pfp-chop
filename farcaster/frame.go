package farcaster

import (
	"fmt"
	"io"
)

type Action int

const (
	ActionPOST Action = iota
	ActionPOSTRedirect
	ActionMint
	ActionLink
)

func (a Action) String() string {
	switch a {
	case ActionPOST:
		return "post"
	case ActionPOSTRedirect:
		return "post_redirect"
	case ActionMint:
		return "mint"
	case ActionLink:
		return "link"
	}
	return "unknown"
}

type UntrustedData struct {
	FID         uint64 `json:"fid"`
	URL         string `json:"url"`
	MessageHash string `json:"messageHash"`
	Timestamp   uint64 `json:"timestamp,uint64"`
	Network     int    `json:"network"`
	ButtonIndex int    `json:"buttonIndex"`
	InputText   string `json:"inputText"`
	CastId      struct {
		FID  uint64 `json:"fid"`
		Hash string `json:"hash"`
	} `json:"castId"`
}

type SignaturePacket struct {
	UntrustedData UntrustedData `json:"untrustedData"`
	TrustedData   struct {
		MessageBytes string `json:"messageBytes"`
	}
}

type Button struct {
	Label  []byte
	Action Action
	Target []byte
}

type Frame struct {
	FrameV         string
	Image          string
	PostURL        string
	Buttons        []Button
	InputTextLabel string
}

func (f *Frame) Render(w io.Writer) {

	btns := ""
	for idx, b := range f.Buttons {
		i := idx + 1
		btns += fmt.Sprintf(`<meta property="fc:frame:button:%d" content="%s">\n`, i, b.Label)
		btns += fmt.Sprintf(`<meta property="fc:frame:button:%d:action" content="%s">\n`, i, b.Action.String())
		btns += fmt.Sprintf(`<meta property="fc:frame:button:%d:target" content="%s">\n`, i, b.Target)
	}

	inputTx := ""
	if f.InputTextLabel != "" {
		inputTx = fmt.Sprintf(`<meta property="fc:frame:input:text" content="%s">\n`, f.InputTextLabel)
	}

	// TODO aspect ratio

	resp := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <title>Frame</title>
  <meta property="og:image" content="%s">
  <meta property="fc:frame" content="%s">
  <meta property="fc:frame:image" content="%s">
  <meta property="fc:frame:post_url" content="%s">
  %s
  %s
</head>
  <body>
    greetings
  </body>
</html>`, f.Image, f.FrameV, f.Image, f.PostURL, btns, inputTx)
	if _, err := w.Write([]byte(resp)); err != nil {
		panic(err)
	}

}
