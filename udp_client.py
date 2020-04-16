#!/usr/bin/env python

from __future__ import print_function
import os
import sys
import time
import json
import socket
import calendar

USER = os.getenv('USER')
HOST, PORT = socket.gethostname(), 9331
client_socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
Nmsg = 2
try:
    Nmsg = int(sys.argv[1])
except:
    pass

print('num messages ', Nmsg)
num_retransmits = 0
seed = calendar.timegm(time.gmtime())
while(num_retransmits < Nmsg):
    num_retransmits += 1
    seed += 1
    data = {'read_vector_bytes': 133206046, 
            'site_name': 'T2_US_Purdue', 
            'read_vector_count_average': 21.411799999999999, 
            'user_dn': '/DC=ch/DC=cern/OU=Organic Units/OU=Users/CN=%s%s' % (USER, seed), 
            'file_lfn': '/store/fake/file_%s.root'% seed, 
            'read_bytes': 148607872, 
            'file_size': 27502730289, 
            'read_single_average': 3793.5500000000002, 
            'client_host': 'rossmann-a251', 
            'read_vector_average': 7835650.0, 
            'read_vector_sigma': 7081190.0, 
            'server_host': 'cmshdp-d019', 
            'read_vector_operations': 17, 
            'read_single_bytes': 15401826, 
            'app_info': 'something', 
            'client_domain': HOST,
            'start_time': 1395960729, 
            'read_vector_count_sigma': 56.063099999999999, 
            'read_single_sigma': 84703.899999999994, 
            'server_domain': HOST,
            'read_single_operations': 4060, 
            'read_bytes_at_close': 148607872, 
            'end_time': 1395960959, 
            'fallback': False, 
            'unique_id': '60DC3A6D-02B6-E311-B2BD-0002C90B73D8-0%s'% seed
            } 

    data = json.dumps(data)
    client_socket.sendto(data, (HOST, PORT))
print("sent all messages")
