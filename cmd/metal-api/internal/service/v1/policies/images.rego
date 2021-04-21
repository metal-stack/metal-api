package api.v1.metalstack.io.authz

e = permissions["metal.v1.image.list"] {
    input.path = ["v1", "image"]
    input.method = "GET"
}

e = permissions["metal.v1.image.get"] {
    some id
    input.path = ["v1", "image", id]
    input.method = "GET"
}

e = permissions["metal.v1.image.get-latest"] {
    some id
    input.path = ["v1", "image", id, "latest"]
    input.method = "GET"
}

e = permissions["metal.v1.image.delete"] {
    some id
    input.path = ["v1", "image", id]
    input.method = "DELETE"
}

e = permissions["metal.v1.image.create"] {
    input.method = "PUT"
    input.path = ["v1", "image"]
}

e = permissions["metal.v1.image.update"] {
    input.method = "POST"
    input.path = ["v1", "image"]
}
