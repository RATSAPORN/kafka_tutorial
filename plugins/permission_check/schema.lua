return {
    name = "permission_check",

    fields = {
        {
            config = {
                type = "record",
                fields = {
                    {
                        permission_url = {
                            type = "string",
                            required = true
                        }
                    },
                    {
                        permission_timeout = {
                            type = "number",
                            default = 1000
                        }
                    },
                    {
                        public_paths = {
                            type = "array",
                            elements = {
                                type = "string"
                            }
                        }
                    }
                }
            }
        }
    }
}