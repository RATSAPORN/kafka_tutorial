local http = require "resty.http"
local cjson = require "cjson"

local PermissionCheck = {
    PRIORITY = 1000,
    VERSION = "1.0.0",
}

-- Check if current path is in the public paths list
local function is_public_path(path, public_paths)
    if not public_paths then
        return false
    end
    for _, public_path in ipairs(public_paths) do
        if path == public_path or string.match(path, public_path) then
            return true
        end
    end
    return false
end

function PermissionCheck:access(conf)
    local request_path = kong.request.get_path()
    
    -- Skip permission check for public paths (like login)
    if is_public_path(request_path, conf.public_paths) then
        kong.log.notice("[permission_check] Public path, skipping permission check: ", request_path)
        return
    end
    kong.log.notice("Updated Hello")
    local auth_header = kong.request.get_header("Authorization")
    local headers = kong.request.get_headers()
    kong.log.notice("[permission_check] All request headers = ", require("cjson.safe").encode(headers))
    local request_data = {
        method = kong.request.get_method(),
        path = request_path,
        authorization = auth_header
    }

    local upstream_headers = {}
    for k, v in pairs(headers) do
        upstream_headers[k] = v
    end

    upstream_headers["Content-Length"] = nil   -- resty.http will recalculate this from the new body
    upstream_headers["content-length"] = nil   -- headers table can be case-insensitive, but be safe
    upstream_headers["Host"] = nil             -- let resty.http set Host for permission_url
    upstream_headers["host"] = nil
    upstream_headers["X-Original-URI"]=conf.permission_url
    upstream_headers["X-Original-Method"]="POST"

    local httpc = http.new()
    kong.log.notice("Authorization header = ", auth_header)
    kong.log.notice("Request body = ", cjson.encode(request_data))
    local res, err = httpc:request_uri(
        conf.permission_url,
        {
            method = "POST",
            body = cjson.encode(request_data),

            headers = upstream_headers,
        }
    )
    
    if not res then
        kong.log.err("Permission service unreachable: ", err)

        return kong.response.exit(500, {
            message = "Permission service unavailable"
        })
    end
    kong.log.notice("Permission status = ", res.status)
    kong.log.notice("Permission response = ", res.body)
    if res.status ~= 200 then
        return kong.response.exit(403, {
            message = "Permission denied"
        })
    end

    local ok, body = pcall(cjson.decode, res.body)

    if not ok then
       return kong.response.exit(500, {
        message = "Invalid permission service response"
       })
    end

    if body.user_id then
       kong.service.request.set_header("X-User-ID", tostring(body.user_id))
    end

    if body.role then
        kong.service.request.set_header(
            "X-User-Role",
            body.role
        )
    end

    kong.log.notice("Permission granted")
end

return PermissionCheck