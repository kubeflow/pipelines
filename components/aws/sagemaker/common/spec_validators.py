class SpecValidators:
    @staticmethod
    def nullable_string_argument(value):
        value = value.strip()
        if not value:
            return None
        return value

    @staticmethod
    def _yaml_or_json_str(str):
        if str == "" or str == None:
            return None
        try:
            return json.loads(str)
        except:
            return yaml.safe_load(str)

    @staticmethod
    def yaml_or_json_list(str):
        return SpecValidators._yaml_or_json_str
    
    @staticmethod
    def yaml_or_json_dict(str):
        return SpecValidators._yaml_or_json_str

    @staticmethod
    def str_to_bool(str):
        # This distutils function returns an integer representation of the boolean
        # rather than a True/False value. This simply hard casts it.
        return bool(strtobool(str))