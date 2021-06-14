from datetime import datetime

dateTimeObj = datetime.now()
timestampStr = dateTimeObj.strftime("%d-%m-%Y-%H-%M-%S.%f")
print(timestampStr, end='')