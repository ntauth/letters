package letters

type (
	ParseOptions struct {
		SkipAttachments bool
	}

	ParseOption func(opts *ParseOptions)
)

func WithParseSkipAttachments() ParseOption {
	return func(opts *ParseOptions) {
		opts.SkipAttachments = true
	}
}
