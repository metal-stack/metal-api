package api.v1.metalstack.io.authz

e = {"permission": permissions["metal.v1.image.list"], "public": false} {
    input.path = ["v1", "image"]
    input.method = "GET"
}

e = {"permission": permissions["metal.v1.image.get"], "public": false} {
    some id
    input.path = ["v1", "image", id]
    input.method = "GET"
}

e = {"permission": permissions["metal.v1.image.get-latest"], "public": false}{
    some id
    input.path = ["v1", "image", id, "latest"]
    input.method = "GET"
}

e = {"permission": permissions["metal.v1.image.delete"], "public": false} {
    some id
    input.path = ["v1", "image", id]
    input.method = "DELETE"
}

e = {"permission": permissions["metal.v1.image.create"], "public": false} {
    input.method = "PUT"
    input.path = ["v1", "image"]
}

e = {"permission": permissions["metal.v1.image.update"], "public": false} {
    input.method = "POST"
    input.path = ["v1", "image"]
}
