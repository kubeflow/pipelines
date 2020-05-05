def check_empty_string_values(obj):
    obj_has_empty_string = 0
    if type(obj) is dict:
        for k,v in obj.items():
            if type(v) is str and v == '':
                print(k + '- has empty string value')
                obj_has_empty_string = 1
            elif type(v) is dict:
                check_empty_string_values(v)

