class SpecValidation:
    @staticmethod
    def _nullable_string_argument(value):
        value = value.strip()
        if not value:
            return None
        return value