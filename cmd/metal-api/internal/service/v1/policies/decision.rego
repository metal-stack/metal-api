package api.v1.metalstack.io.authz

default decision = {"allow": false, "isAdmin": false}

admin(e) {
	ae := sprintf("%s.admin", [e])
    permissions[ae]
    input.permissions[ae]
}

user(e) {
	input.permissions[e]
}

decision = {"allow": true, "isAdmin": false} {
	user(e)
    not admin(e)
}

decision = {"allow": true, "isAdmin": true} {
    admin(e)
}

decision = {"allow": false, "isAdmin": false, "reason": reason} {
    not user(e)
    not admin(e)
    reason := sprintf("missing permission on %s", [e])
}
