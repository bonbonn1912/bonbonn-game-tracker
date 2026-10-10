#!/usr/bin/env python3
"""Compare legacy and Go instances against the same deterministic FACEIT fixtures.
Never point this at production: it opens and closes a synthetic match.
LEGACY_URL=http://127.0.0.1:19081 GO_URL=http://127.0.0.1:19082 python3 scripts/compatibility.py
"""
import json
import os
import urllib.error
import urllib.request

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args):
        return None

opener = urllib.request.build_opener(NoRedirect)
key = '4967963b-566f-4e8a-908b-02c8315d12f1'
unknown = '11111111-1111-1111-1111-111111111111'
bases = [os.environ['LEGACY_URL'], os.environ['GO_URL']]

def request(base, method, path, body=None, headers=None):
    h = {'Accept': '*/*', **(headers or {})}
    data = None
    if body is not None:
        data = body if isinstance(body, bytes) else json.dumps(body).encode()
        h.setdefault('Content-Type', 'application/json')
    req = urllib.request.Request(base + path, data=data, method=method, headers=h)
    try:
        response = opener.open(req, timeout=15)
    except urllib.error.HTTPError as e:
        response = e
    raw = response.read().decode()
    content_type = response.headers.get('Content-Type', '')
    if raw and 'application/json' in content_type:
        raw = json.loads(raw)
    return response.status, raw, content_type, response.headers.get('Location'), response.headers.get('Access-Control-Allow-Origin')

cases = [(m, p, None, None) for m,p in [
    ('GET','/test'),('POST','/test'),('GET','/unknown'),
    ('GET','/elo'),('GET','/elo?username=a'),('GET','/elo?username=abc%20def'),
    ('GET','/elobyid'),('GET','/elobysteamid?id=1'),
    ('GET','/matchhistory?username=abc&limit=3'),('GET','/matchhistory?username=abc&limit=101'),
    ('GET','/getmatch'),('GET','/redirect?key=BAD'),
    ('GET',f'/getmatch?key={key}'),('GET',f'/redirect?key={key}'),
    ('GET','/elo?username=BonBonn-_-'),('GET','/checkelo?username=BonBonn-_-'),
    ('GET',f'/elobyid?id={key}'),('GET','/elobysteamid?id=76561198000000000'),
    ('GET','/elo?username=NoCS2'),('GET','/elo?username=invalid'),
    ('GET','/matchhistory?username=BonBonn-_-&limit=5'),
    ('GET','/ELO?username=BonBonn-_-'),('GET','/elo/?username=BonBonn-_-'),
    ('OPTIONS','/elo'),
    ('HEAD','/test'),('HEAD','/elo?username=BonBonn-_-'),
    ('GET','/elo?username=BonBonn-_-&id='),
    ('GET','/elo?username=BonBonn-_-&username=NoCS2'),
    ('GET','/matchhistory?username=BonBonn-_-&limit=5.5'),
    ('GET','/matchhistory?username=BonBonn-_-&limit=%205%20'),
    ('GET','/matchhistory?username=BonBonn-_-&limit=0x10'),
]]
cases.append(('POST','/internal/api/getMultiplePlayers',{'players':[{'nickname':'BonBonn-_-','steam64ID':'76561198000000000','custom':'preserve'}]},None))
body = {'event':'match_status_configuring','event_id':'compatibility-event','custom':'preserve','payload':{'id':'compatibility-match','entity':{'type':'matchmaking'},'teams':[{'name':'Alpha','roster':[{'nickname':'Player'+str(i)} for i in range(5)]},{'name':'Beta','roster':[{'nickname':'Player'+str(i+5)} for i in range(5)]}]}}
auth = {'Authorization':key}
cases.extend([
    ('POST','/match',body,None),
    ('POST','/match',{**body,'payload':{**body['payload'],'entity':{'type':'tournament'}}},auth),
    ('POST','/match',{**body,'event':'match_object_created'},auth),
    ('POST','/match',body,auth),
    ('POST','/match',body,auth),
    ('GET',f'/getmatch?key={key}',None,None),
    ('GET',f'/redirect?key={key}',None,None),
    ('GET',f'/redirect?key={key}',None,{'Accept':'text/html'}),
    ('GET',f'/redirect?key={key}',None,{'Accept':'application/json'}),
    ('HEAD',f'/redirect?key={key}',None,None),
    ('POST','/match',{**body,'event':'match_status_finished'},auth),
    ('GET',f'/getmatch?key={key}',None,None),
    ('GET',f'/redirect?key={key}',None,None),
    ('POST','/match',body,{'Authorization':unknown}),
    ('GET',f'/getmatch?key={unknown}',None,None),
    ('POST','/match',{**body,'event':'match_status_finished'},{'Authorization':unknown}),
])
cases.extend([
    ('POST','/internal/api/getMultiplePlayers',b'players[][nickname]=BonBonn-_-&players[][steam64ID]=76561198000000000',{'Content-Type':'application/x-www-form-urlencoded'}),
    ('POST','/internal/api/getMultiplePlayers',b'players[0][nickname]=BonBonn-_-&players[2][nickname]=NoCS2',{'Content-Type':'application/x-www-form-urlencoded'}),
])
failures = 0
for method,path,body,headers in cases:
    results = [request(base,method,path,body,headers) for base in bases]
    if path == '/internal/api/getMultiplePlayers':
        # Legacy Promise callbacks append in completion order, not input order.
        results = [(status, sorted(body, key=lambda p: str(p.get('nickname',''))), *rest) if isinstance(body,list) else (status,body,*rest) for status,body,*rest in results]
    ok = results[0] == results[1]
    print(('PASS' if ok else 'FAIL'),method,path)
    if not ok:
        failures += 1
        print('  legacy:',repr(results[0]))
        print('  go:    ',repr(results[1]))
print(f'{len(cases)-failures}/{len(cases)} compatibility cases passed')
raise SystemExit(bool(failures))
