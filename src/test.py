from datetime import datetime

f = open("demofile2.txt", "a")
f.writelines([datetime.now().strftime("%m/%d/%Y, %H:%M:%S")])
f.close()

print("file super written")