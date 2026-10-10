import json
from http.server import ThreadingHTTPServer,BaseHTTPRequestHandler
from urllib.parse import urlparse,parse_qs
PLAYER='4967963b-566f-4e8a-908b-02c8315d12f1'
class Handler(BaseHTTPRequestHandler):
 def log_message(self,*args): pass
 def do_GET(self):
  u=urlparse(self.path);q=parse_qs(u.query)
  if u.path.endswith('/history'):
   data={'items':[{'results':{'winner':'faction1' if w else 'faction2'},'teams':{'faction1':{'players':[{'player_id':PLAYER}]},'faction2':{'players':[{'player_id':'enemy'}]}}} for w in [True,False,True,True,False]]}
  elif u.path.startswith('/matches/'):
   data={'voting':{'map':{'pick':['de_mirage']}}}
  elif q.get('nickname')==['invalid']:
   data={'errors':[{'message':'not found'}]}
  else:
   data={'player_id':PLAYER,'nickname':q.get('nickname',['BonBonn-_-'])[0],'games':{'csgo':{'faceit_elo':1500},'cs2':{'faceit_elo':1600}},'platforms':{'steam':'76561198000000000'}}
   if q.get('nickname')==['NoCS2']: del data['games']['cs2']
  raw=json.dumps(data).encode();self.send_response(200);self.send_header('Content-Type','application/json');self.end_headers();self.wfile.write(raw)
ThreadingHTTPServer(('127.0.0.1',19083),Handler).serve_forever()
