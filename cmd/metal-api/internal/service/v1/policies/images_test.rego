package api.v1.metalstack.io.authz

test_image_list_allowed {
    decision.allow with input as {"path": ["v1", "image"], "method": "GET", "permissions": {"metal.v1.image.get": true, "metal.v1.image.list": true}}
}

test_image_list_not_allowed_without_permissions {
    not decision.allow with input as {"path": ["v1", "image"], "method": "GET", "permissions": {}}
}

test_image_list_not_allowed_without_fitting_permissions {
    not decision.allow with input as {"path": ["v1", "image"], "method": "GET", "permissions": {"metal.v1.image.get": true}}
    # FIXME not working somehow
    # decision.reason = "missing permission on metal.v1.image.list"
}

test_image_get_allowed {
    decision.allow with input as {"path": ["v1", "image", "ubuntu-20.04"], "method": "GET", "permissions": {"metal.v1.image.get": true, "metal.v1.image.list": true}}
}

test_image_get_not_allowed_without_permissions {
    not decision.allow with input as {"path": ["v1", "image", "ubuntu-20.04"], "method": "GET", "permissions": {}}
}

test_image_get_not_allowed_without_fitting_permissions {
    not decision.allow with input as {"path": ["v1", "image", "ubuntu-20.04"], "method": "GET", "permissions": {"metal.v1.image.list": true}}
}

test_image_get_latest_allowed {
    decision.allow with input as {"path": ["v1", "image", "ubuntu-20.04", "latest"], "method": "GET", "permissions": {"metal.v1.image.get-latest": true}}
}

test_image_get_latest_not_allowed_without_permissions {
    not decision.allow with input as {"path": ["v1", "image", "ubuntu-20.04", "latest"], "method": "GET", "permissions": {}}
}

test_image_get_latest_not_allowed_without_fitting_permissions {
    not decision.allow with input as {"path": ["v1", "image", "ubuntu-20.04", "latest"], "method": "GET", "permissions": {"metal.v1.image.get": true}}
}

test_image_delete_allowed {
    decision.allow with input as {"path": ["v1", "image", "ubuntu-20.04"], "method": "DELETE", "permissions": {"metal.v1.image.delete": true}}
}

test_image_delete_not_allowed_without_permissions {
    not decision.allow with input as {"path": ["v1", "image", "ubuntu-20.04"], "method": "DELETE", "permissions": {}}
}

test_image_delete_not_allowed_without_fitting_permissions {
    not decision.allow with input as {"path": ["v1", "image", "ubuntu-20.04"], "method": "DELETE", "permissions": {"metal.v1.image.get": true}}
}

test_image_create_allowed {
    decision.allow with input as {"path": ["v1", "image"], "method": "PUT", "permissions": {"metal.v1.image.create": true}}
}

test_image_create_not_allowed_without_permissions {
    not decision.allow with input as {"path": ["v1", "image"], "method": "PUT", "permissions": {}}
}

test_image_create_not_allowed_without_fitting_permissions {
    not decision.allow with input as {"path": ["v1", "image"], "method": "PUT", "permissions": {"metal.v1.image.get": true}}
}

test_image_update_allowed {
    decision.allow with input as {"path": ["v1", "image"], "method": "POST", "permissions": {"metal.v1.image.update": true}}
}

test_image_update_not_allowed_without_permissions {
    not decision.allow with input as {"path": ["v1", "image"], "method": "POST", "permissions": {}}
}

test_image_update_not_allowed_without_fitting_permissions {
    not decision.allow with input as {"path": ["v1", "image"], "method": "POST", "permissions": {"metal.v1.image.get": true}}
}
