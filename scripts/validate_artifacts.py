#!/usr/bin/env python3
"""Offline integrity, coverage, PAD metadata and local-link checks. No writes."""
from pathlib import Path
import hashlib,json,re,sys
from urllib.parse import unquote,urlsplit
import yaml
ROOT=Path(__file__).resolve().parents[1]
errors=[]; checks=0
def check(ok,msg):
 global checks
 checks+=1
 if not ok:errors.append(msg)
def digest(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def meta(p):
 s=p.read_text(encoding='utf-8')
 if s.startswith('---\n'):return yaml.safe_load(s.split('---',2)[1])
 return None
def main():
 ref=ROOT/'docs/reference'; manifest=json.loads((ref/'manifest.json').read_text())
 for f in manifest['files']:
  p=ref/f['path'];check(p.is_file(),f'Missing original: {p}')
  if p.is_file():check(p.stat().st_size==f['bytes'] and digest(p)==f['sha256'],f'Changed original: {p}')
 for source,target in [('schema.sql','docs/database/schema.sql'),('schema_tests.sql','tests/database/schema_tests.sql'),('schema_race_tests.sh','tests/database/schema_race_tests.sh')]:
  check((ref/'originals'/source).read_bytes()==(ROOT/target).read_bytes(),f'Baseline changed: {target}')
 coverage=json.loads((ref/'coverage.json').read_text());original=(ref/coverage['source']).read_text();parts=re.split(r'(?m)(?=^## )',original)
 check(sorted(s['part'] for s in coverage['sections'])==list(range(len(parts))),'Coverage does not account for every source section exactly once')
 for s in coverage['sections']:
  part=parts[s['part']];check(hashlib.sha256(part.encode()).hexdigest()==s['sha256'],'Section hash mismatch')
  check(part in (ROOT/s['target']).read_text(),f'Full original section missing: {s["target"]} part {s["part"]}')
 # Validate PAD YAML/front matter against the template enums and linked identities.
 for p in [ROOT/'pad.yaml',ROOT/'.agent/state/current.yaml']:
  try:check(isinstance(yaml.safe_load(p.read_text()),dict),f'Invalid YAML: {p}')
  except Exception as e:check(False,f'YAML {p}: {e}')
 tasks={};reqs={};adrs={}
 for folder,schemafile,store,skip in [('tasks','task.schema.yaml',tasks,'TASK_TEMPLATE.md'),('requirements','requirement.schema.yaml',reqs,'REQUIREMENT_TEMPLATE.md')]:
  schema=yaml.safe_load((ROOT/'.agent/schemas'/schemafile).read_text())
  for p in sorted((ROOT/'.agent'/folder).glob('*.md')):
   if p.name in ['README.md',skip]:continue
   m=meta(p);check(isinstance(m,dict),f'Missing metadata {p}')
   if not isinstance(m,dict):continue
   check(all(k in m for k in schema['required_front_matter']),f'Missing required fields {p}')
   check(re.fullmatch(schema['fields']['id'],str(m.get('id',''))) is not None,f'Bad id {p}')
   for key,allowed in schema['fields'].items():
    if key in m and isinstance(allowed,list):check(m[key] in allowed,f'Invalid {key} in {p}')
   check(m['id'] not in store,f'Duplicate identity {m["id"]}');store[m['id']]=m
 for p in (ROOT/'.agent/decisions').glob('ADR-*.md'):
  if p.name.endswith('-template.md'):continue
  m=meta(p);check(bool(m) and all(k in m for k in ['id','title','status','date','deciders']),f'Invalid ADR {p}')
  if m:check(m['id'] not in adrs,f'Duplicate ADR {m["id"]}');adrs[m['id']]=m
 for id,t in tasks.items():
  for dep in t.get('depends_on',[]):check(dep in tasks,f'Unknown task dependency {dep}')
  for req in t.get('requirement_ids',[]):check(req in reqs,f'Unknown requirement {req}')
 visiting=set();done=set()
 def visit(id):
  if id in visiting:check(False,f'Task cycle at {id}');return
  if id in done:return
  visiting.add(id)
  for dep in tasks[id].get('depends_on',[]):
   if dep in tasks:visit(dep)
  visiting.remove(id);done.add(id)
 for id in tasks:visit(id)
 check(len([t for t in tasks.values() if any(x.startswith('stage-') for x in t.get('labels',[]))])==12,'All 12 roadmap stages must have tasks')
 # Local Markdown links, ignoring fenced code (DDL is documentation, not links).
 for p in sorted(ROOT.rglob('*.md')):
  if any(x in p.relative_to(ROOT).parts for x in ['.git','node_modules']):continue
  body=re.sub(r'(?ms)^```.*?^```[^\n]*\n?','',p.read_text())
  for target in re.findall(r'!?\[[^\]]*\]\(([^)]+)\)',body):
   target=target.strip().split(' "',1)[0].strip('<>')
   parsed=urlsplit(target)
   if parsed.scheme or parsed.netloc or not parsed.path:continue
   dest=p.parent/unquote(parsed.path)
   check(dest.exists(),f'Broken local link in {p.relative_to(ROOT)}: {target}')
 # Template managed baseline must remain untouched: policy docs match upstream baseline.
 for directory in ['protocols','agents','schemas']:
  base=ROOT/'.pad/template-base/.agent'/directory
  for p in base.rglob('*'):
   if p.is_file():check(p.read_bytes()==(ROOT/'.agent'/directory/p.relative_to(base)).read_bytes(),f'Original PAD {directory} changed: {p.name}')
 if errors:
  print('\n'.join('FAIL: '+e for e in errors));print(f'{len(errors)} failure(s), {checks} checks');return 1
 print(f'PASS: {checks} integrity/coverage/metadata/link checks; 4 originals; 12 stages; no lost source sections');return 0
if __name__=='__main__':sys.exit(main())
