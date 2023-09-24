package dbaccess

// FakeCloser is a test fake for Closer.
type FakeCloser struct{}

// Close implements the Closer interface on FakeCloser.
func (c *FakeCloser) Close() {}

// FakeUserInserter is a test fake for Inserter[User].
type FakeUserInserter struct{ OutErr error }

// Insert implements the Inserter[User] interface on FakeUserInserter.
func (f *FakeUserInserter) Insert(_ User) error { return f.OutErr }

// FakeUserSelector is a test fake for Selector[User].
type FakeUserSelector struct {
	OutRes User
	OutErr error
}

// Select implements the Selector[User] interface on FakeUserSelector.
func (f *FakeUserSelector) Select(_ string) (User, error) {
	return f.OutRes, f.OutErr
}

// FakeCounter is a test fake for Counter.
type FakeCounter struct {
	OutRes int
	OutErr error
}

// Count implements the Counter interface on FakeCounter.
func (f *FakeCounter) Count(_ string) (int, error) { return f.OutRes, f.OutErr }

// FakeBoardSelector is a test fake for Selector[Board].
type FakeBoardSelector struct{ OutErr error }

// Select implements the Selector[Board] interface on FakeBoardSelector.
func (f *FakeBoardSelector) Select(_ string) (Board, error) {
	return Board{}, f.OutErr
}

// FakeBoardInserter is a test fake for Inserter[InBoard].
type FakeBoardInserter struct{ OutErr error }

// Insert implements the Inserter[InBoard] interface on FakeBoardInserter.
func (f *FakeBoardInserter) Insert(_ InBoard) error { return f.OutErr }

// FakeUserBoardSelector is a test fake for RelSelector[bool].
type FakeUserBoardSelector struct {
	OutIsAdmin bool
	OutErr     error
}

// Select implements the RelSelector[bool] interface on FakeUserBoardSelector.
func (f *FakeUserBoardSelector) Select(_, _ string) (bool, error) {
	return f.OutIsAdmin, f.OutErr
}

// FakeDeleter is a test fake for Deleter.
type FakeDeleter struct{ OutErr error }

// Delete implements the Deleter interface on FakeDeleter.
func (f *FakeDeleter) Delete(_ string) error { return f.OutErr }
