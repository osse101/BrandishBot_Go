import sys
from collections import defaultdict

log_file = sys.argv[1]

user_jobs = defaultdict(lambda: defaultdict(int))
user_names = {}

with open(log_file, 'r') as f:
    for line in f:
        # Extract user_id and username mappings
        if 'user_id=' in line and 'username=' in line:
            parts = line.split()
            uid = ""
            uname = ""
            for p in parts:
                if p.startswith('user_id='): uid = p.split('=', 1)[1]
                if p.startswith('username='): uname = p.split('=', 1)[1]
            if uid and uname:
                user_names[uid] = uname
                
        # Count XP awards
        if 'msg="Awarded job XP"' in line:
            parts = line.split()
            uid = ""
            job = ""
            for p in parts:
                if p.startswith('user_id='): uid = p.split('=', 1)[1]
                if p.startswith('job='): job = p.split('=', 1)[1]
            if uid and job:
                user_jobs[uid][job] += 1

print("Log Analysis:")
print(f"{'Username':<20} | {'Scholar (Engage)':<18} | {'Explorer (Search)':<18} | {'Farmer (Harvest)':<18}")
print("-" * 80)

for uid, jobs in user_jobs.items():
    uname = user_names.get(uid, uid)
    scholar = jobs.get('job_scholar', 0)
    explorer = jobs.get('job_explorer', 0)
    farmer = jobs.get('job_farmer', 0)
    print(f"{uname:<20} | {scholar:<18} | {explorer:<18} | {farmer:<18}")

print("\nFinished.")
