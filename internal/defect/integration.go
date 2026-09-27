package defect

func Healthy(store *Store) bool { return store != nil && store.defects != nil }
