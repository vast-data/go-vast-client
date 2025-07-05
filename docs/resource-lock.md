In concurrent applications, it's common for multiple operations 
to target the same backend resource at the same time — 
such as modifying the same user, group, or bucket. To prevent race conditions and ensure consistency, 
the vast_client provides a built-in resource locking mechanism.

You can use resource locks to:

- Ensures that only one operation at a time can modify a specific resource.
- Prevents conflicts when working with shared data.
- Helps maintain API consistency during complex or multi-step operations.

Call the `Lock()` method on a resource before performing operations that should not overlap. 
Always use defer to ensure the lock is released automatically:

```go
defer rest.Users.Lock("uid", 3001)()
// safely perform operations on user with uid=3001
```

Lock using a specific key (e.g. ID, name):
```go
defer rest.Groups.Lock("gid", 1001)()
```

Lock using multiple keys (for composite identity):
```go
defer rest.Quotas.Lock("tenant", 1, "path", "/shared")()
```

Lock with no key (acts as a general-purpose lock using an empty string):
```go
defer rest.Users.Lock()()
```

