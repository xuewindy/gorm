# [GORM](https://github.com/jinzhu/gorm)

[![Go Report Card](https://goreportcard.com/badge/github.com/sljeff/gorm)](https://goreportcard.com/report/github.com/sljeff/gorm)
[![wercker status](https://app.wercker.com/status/c8794d29309d12e6f3b52d177bd1e644/s/master "wercker status")](https://app.wercker.com/project/byKey/c8794d29309d12e6f3b52d177bd1e644)
[![codecov](https://codecov.io/gh/sljeff/gorm/branch/master/graph/badge.svg)](https://codecov.io/gh/sljeff/gorm)
[![Join the chat at https://gitter.im/jinzhu/gorm](https://img.shields.io/gitter/room/jinzhu/gorm.svg)](https://gitter.im/jinzhu/gorm?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![Open Collective Backer](https://opencollective.com/gorm/tiers/backer/badge.svg?label=backer&color=brightgreen "Open Collective Backer")](https://opencollective.com/gorm)
[![Open Collective Sponsor](https://opencollective.com/gorm/tiers/sponsor/badge.svg?label=sponsor&color=brightgreen "Open Collective Sponsor")](https://opencollective.com/gorm)
[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://godoc.org/github.com/jinzhu/gorm?status.svg)](https://godoc.org/github.com/jinzhu/gorm)

## Overview

### Remove Support for pgsql 9.3 and 9.4

Because of `ON CONFLICT`.

PostgreSQL not support ON CONFLICT IGNORE (`ON CONFLICT DO NOTHING`) because gorm can not get right `RowsAffected`.

### GetOrCreate

```go
// Logic: get => create => get again when create failed
db.Where(User{Name: "jinzhu"}).Attrs(User{Age: 30}).GetOrCreate(&user)
```

### IGNORE/ON CONFLICT UPDATE

```go
// mysql: INSERT IGNORE INTO
// sqlite: INSERT OR IGNORE
db.CreateOnConflict(User{UserName: "gorm"}, gorm.IGNORE)

// mysql: INSERT INTO ... ON DUPLICATE KEY UPDATE ...
db.CreateOnConflict(User{UserName: "gorm"}, User{LastLoginAt: time.Now()})

// postgresql: INSERT INTO ... ON CONFLICT ON CONSTRAINT constraint_name DO UPDATE ...
db.CreateOnConflict(User{UserName: "gorm"}, "constraint_name", User{LastLoginAt: time.Now()})
```

### CreateMany/CreateMany OnConflict

```go
// mysql and sqlite: insert multiple; insert multiple ignore duplicate
// postgresql and mssql do not support ignore
db.CreateMany([]interface{}{&user1, &user2, &user3}, gorm.IGNORE)
db.CreateMany([]interface{}{&user1, &user2, &user3})

// mysql: insert on conflict update
db.CreateMany([]interface{}{&user1, &user2, &user3}, &User{UpdatedAt: now})

// postgresql: insert on confilct update
db.CreateMany([]interface{}{&user1, &user2, &user3}, 'constraint_name', &User{UpdatedAt: now})

// Caution: mssql db driver will not raise error on duplicate
db.CreateMany([]interface{}{&user1, &user1, &user1})
```

> Caution: mssql db driver will not raise error on duplicate
> **Caution: CreateMany will not trigger `afterCreateCallback`**

### MySQL VALUES function

```go
// only support MySQL: ON DUPLICATE KEY UPDATE `field` = VALUES(`field`), ...
DB.CreateMany([]interface{}{
	&Email{Id: 1, UserId: 100, Email: "jeff@example.com"},
	&Email{Id: 2, UserId: 100, Email: "alan@example.com"},
}, "updated_at", "email")
db.CreateOnConflict(User{UserName: "gorm"}, "updated_at")
```

## License

© Jinzhu, 2013~time.Now

Released under the [MIT License](https://github.com/jinzhu/gorm/blob/master/License)
