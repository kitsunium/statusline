// Written once from the design; this file is yours. The shells in
// gen_design.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package quota

// resolve lets stdin win wherever both sources carry a bucket: it is free,
// synchronous and always current, while the API is cached and slightly
// behind. The API only contributes the model-scoped quotas, the credit
// balance and the buckets a host build did not send. A bucket missing from
// both stays missing: rendering it as 0 % would lie about the account.
func resolve(stdin, api Set) Set {
	merged := stdin
	if !merged.Session.IsValid() && api.Session.IsValid() {
		merged.Session = api.Session
	}
	if !merged.Weekly.IsValid() && api.Weekly.IsValid() {
		merged.Weekly = api.Weekly
	}
	merged.Scoped = api.Scoped
	merged.Extra = api.Extra
	return merged
}
