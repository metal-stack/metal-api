package api.v1.metalstack.io.authz

default allow = false

allow {
    input.path = ["v1", "image"]
    input.method = "GET"

    _ = input.permissions["metal.v1.image.list"]
}

allow {
    some id
    input.path = ["v1", "image", id]
    input.method = "GET"

    _ = input.permissions["metal.v1.image.get"]
}

allow {
    some id
    input.path = ["v1", "image", id, "latest"]
    input.method = "GET"

    _ = input.permissions["metal.v1.image.get-latest"]
}

allow {
    some id
    input.path = ["v1", "image", id]
    input.method = "DELETE"

    _ = input.permissions["metal.v1.image.delete"]
}

allow {
    input.method = "PUT"
    input.path = ["v1", "image"]

    _ = input.permissions["metal.v1.image.create"]
}

allow {
    input.method = "POST"
    input.path = ["v1", "image"]

    _ = input.permissions["metal.v1.image.update"]
}
