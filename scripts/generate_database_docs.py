#!/usr/bin/env python3
"""Deterministic, offline AST projection of the authoritative PostgreSQL DDL.
No database access or SQL execution. Run from any directory; --check never writes.
"""
from pathlib import Path
from collections import Counter
import argparse, hashlib, json, re, sys
from pglast import parse_sql, ast
from pglast.stream import RawStream
ROOT=Path(__file__).resolve().parents[1]
SQL=ROOT/'docs/database/schema.sql'
def strings(xs): return [x.sval for x in (xs or [])]
def raw(node): return RawStream()(node)
def quote(s): return json.dumps(s,ensure_ascii=False)
def build():
    sql=SQL.read_text(encoding='utf-8'); encoded=sql.encode(); parsed=parse_sql(sql)
    tables={}; global_objects=[]; counts=Counter(); foreign=[]
    for rs in parsed:
        st=rs.stmt; counts[type(st).__name__]+=1
        if isinstance(st,ast.CreateStmt):
            name=st.relation.relname
            # SQL statement locations are UTF-8 byte offsets.
            start=rs.stmt_location; end=start+rs.stmt_len if rs.stmt_len else len(encoded)
            ddl=encoded[start:end].decode().strip()+';'
            line=encoded[:start].decode().count('\n')+1
            tables[name]={'name':name,'line':line,'columns':[], 'constraints':[], 'ddl':ddl, 'extras':[]}
            for elt in st.tableElts:
                if isinstance(elt,ast.ColumnDef):
                    cons=list(elt.constraints or [])
                    col={'name':elt.colname,'type':raw(elt.typeName),'ddl':raw(elt),
                         'not_null':any(c.contype.name in ('CONSTR_NOTNULL','CONSTR_PRIMARY') for c in cons),
                         'pk':any(c.contype.name=='CONSTR_PRIMARY' for c in cons),
                         'unique':any(c.contype.name=='CONSTR_UNIQUE' for c in cons),
                         'default':next((raw(c.raw_expr) for c in cons if c.contype.name=='CONSTR_DEFAULT'),None),
                         'notes':[]}
                    tables[name]['columns'].append(col)
                    for c in cons:
                        if c.contype.name=='CONSTR_FOREIGN':
                            foreign.append(dict(table=name,columns=[elt.colname],target=c.pktable.relname,target_columns=strings(c.pk_attrs),delete=c.fk_del_action,update=c.fk_upd_action))
                        if c.contype.name=='CONSTR_CHECK': col['notes'].append('CHECK '+raw(c.raw_expr))
                elif isinstance(elt,ast.Constraint): tables[name]['constraints'].append(elt)
        elif isinstance(st,ast.AlterTableStmt):
            tables[st.relation.relname]['extras'].append(raw(st)+';')
            for cmd in st.cmds:
                if isinstance(cmd.def_,ast.Constraint): tables[st.relation.relname]['constraints'].append(cmd.def_)
        elif isinstance(st,ast.IndexStmt): tables[st.relation.relname]['extras'].append(raw(st)+';')
        elif isinstance(st,ast.CreateTrigStmt): tables[st.relation.relname]['extras'].append(raw(st)+';')
        else:
            start=rs.stmt_location; end=start+rs.stmt_len if rs.stmt_len else len(encoded)
            global_objects.append(encoded[start:end].decode().strip()+';')
    for name,t in tables.items():
        for c in t['constraints']:
            if c.contype.name=='CONSTR_FOREIGN': foreign.append(dict(table=name,columns=strings(c.fk_attrs),target=c.pktable.relname,target_columns=strings(c.pk_attrs),delete=c.fk_del_action,update=c.fk_upd_action))
            if c.contype.name=='CONSTR_PRIMARY':
                for col in t['columns']:
                    if col['name'] in strings(c.keys): col['pk']=True;col['not_null']=True
    # Deduplicate physical FKs; ALTER can add a stricter composite FK in addition to a single FK.
    foreign=list({(f['table'],tuple(f['columns']),f['target'],tuple(f['target_columns'])):f for f in foreign}.values())
    for f in foreign:
        if not f['target_columns']: f['target_columns']=[c['name'] for c in tables[f['target']]['columns'] if c['pk']]
        assert len(f['columns'])==len(f['target_columns']),f
        for c in f['columns']: assert c in [x['name'] for x in tables[f['table']]['columns']],f
        for c in f['target_columns']: assert c in [x['name'] for x in tables[f['target']]['columns']],f
    dbml=['// GENERATED from docs/database/schema.sql. DO NOT EDIT.\n// Lossy visualization; never use DBML as a migration or round-trip SQL source.\nProject FoundryMarches {\n  database_type: \'PostgreSQL\'\n  Note: \'Canonical SQL contains CHECK, EXCLUDE, partial/expression/NULLS NOT DISTINCT indexes, functions, triggers and seed data not executable here. See objects.md and tables.md.\'\n}\n']
    md=['# Полный каталог таблиц\n\nГенерируется из `schema.sql`. Здесь все колонки и исходный DDL; поздние ALTER, индексы и явные триггеры включены. Динамические триггеры и полные функции — в [objects.md](objects.md).\n']
    er=['%% GENERATED from schema.sql. All physical tables/FKs; key columns only.\n%% Conservative many-to-one; UNIQUE can further narrow cardinality.\nerDiagram']
    for name,t in tables.items():
        cols=t['columns']; fks={c for f in foreign if f['table']==name for c in f['columns']}
        pk=[c['name'] for c in cols if c['pk']]
        dbml.append(f'Table {name} {{')
        for c in cols:
            settings=[]
            if c['pk'] and len(pk)==1: settings.append('pk')
            if c['not_null']: settings.append('not null')
            if c['unique']: settings.append('unique')
            if c['default'] is not None: settings.append('default: `'+c['default'].replace('`','')+'`')
            notes=c['notes'][:]
            if c['type']=='visibility': notes.append('PostgreSQL domain visibility: draft, gm_only, roles, public; see SQL')
            if notes: settings.append('note: '+quote('; '.join(notes)))
            # DBML quoted PostgreSQL types support timestamp with time zone / arrays / numeric.
            dbml.append(f'  {c["name"]} {quote(c["type"])}'+(' ['+', '.join(settings)+']' if settings else ''))
        indexes=[]
        if len(pk)>1: indexes.append('    ('+', '.join(pk)+') [pk]')
        for c in t['constraints']:
            if c.contype.name=='CONSTR_UNIQUE':
                settings='unique'; note=[]
                if c.nulls_not_distinct: note.append('SQL UNIQUE NULLS NOT DISTINCT (not represented by DBML semantics)')
                if c.deferrable: note.append('DEFERRABLE; see SQL')
                if note: settings+=', note: '+quote('; '.join(note))
                indexes.append('    ('+', '.join(strings(c.keys))+') ['+settings+']')
        if indexes: dbml+=['  indexes {']+indexes+['  }']
        dbml+=['  Note: '+quote('Canonical definition and PostgreSQL-specific indexes/constraints/triggers: tables.md#'+name+'; full functions and dynamic triggers: objects.md'),'}','']
        er.append('  '+name+' {')
        for c in cols:
            if c['pk'] or c['name'] in fks:
                flags=[]
                if c['pk']:flags.append('PK')
                if c['name'] in fks:flags.append('FK')
                er.append('    '+re.sub('[^A-Za-z0-9_]','_',c['type'])+' '+c['name']+' '+','.join(flags))
        er.append('  }')
        md.append(f'- [{name}](#{name})')
    md.append('\n---\n')
    for name,t in tables.items():
        md.append(f'## {name}\n\n[Источник SQL](schema.sql) · {len(t["columns"])} колонок.\n\n| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |\n|---|---|---|---|---|---|')
        for c in t['columns']:
            md.append('| '+ ' | '.join([f'`{c["name"]}`',f'`{c["type"]}`','нет' if c['not_null'] else 'да','да' if c['pk'] else '—','да' if c['unique'] else '—',('`'+c['default'].replace('|','\\|')+'`') if c['default'] else '—'])+' |')
        md.append('\n### Полный DDL и комментарии\n\n```sql\n'+t['ddl']+'\n```\n')
        if t['extras']:md.append('### Поздние ограничения, индексы и явные триггеры\n\n```sql\n'+'\n'.join(t['extras'])+'\n```\n')
    action={'a':'no action','r':'restrict','c':'cascade','n':'set null','d':'set default','\x00':'no action'}
    for i,f in enumerate(foreign,1):
        left=f['table']+'.'+('('+', '.join(f['columns'])+')' if len(f['columns'])>1 else f['columns'][0])
        right=f['target']+'.'+('('+', '.join(f['target_columns'])+')' if len(f['target_columns'])>1 else f['target_columns'][0])
        dbml.append(f'Ref fk_{i}: {left} > {right} [delete: {action[f["delete"]]}, update: {action[f["update"]]}]')
        # Referenced row optional if any child column is nullable (MATCH SIMPLE).
        nullable=any(not c['not_null'] for c in tables[f['table']]['columns'] if c['name'] in f['columns'])
        er.append('  '+f['target']+(' |o' if nullable else ' ||')+'..o{ '+f['table']+' : "'+','.join(f['columns'])+'"')
    inventory={'schema_sha256':hashlib.sha256(encoded).hexdigest(),'statement_counts':dict(counts), 'table_count':len(tables),'column_count':sum(len(t['columns']) for t in tables.values()),'foreign_key_count':len(foreign),'foreign_keys':foreign,'tables':[]}
    for t in tables.values():
        inventory['tables'].append({k:v for k,v in t.items() if k!='constraints'})
    objects='# Остальные объекты SQL\n\nГенерируемая полная выборка: расширение, домен, функции, DO (в том числе динамические триггеры), seed-данные. Табличный DDL и явные CREATE TRIGGER — в [tables.md](tables.md). Это документация, не отдельный файл миграции.\n\n```sql\n'+'\n\n'.join(global_objects)+'\n```\n'
    return {'docs/database/schema.dbml':'\n'.join(dbml)+'\n','docs/database/er-diagram.mmd':'\n'.join(er)+'\n','docs/database/tables.md':'\n'.join(md)+'\n','docs/database/objects.md':objects,'docs/database/inventory.json':json.dumps(inventory,ensure_ascii=False,indent=2)+'\n'}
def main():
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--check',action='store_true');args=parser.parse_args();changed=[]
    for name,content in build().items():
        content=content.rstrip()+"\n"
        p=ROOT/name
        if args.check:
            if not p.exists() or p.read_text(encoding='utf-8')!=content: changed.append(name)
        else:p.write_text(content,encoding='utf-8')
    if changed:print('STALE:',*changed,sep='\n');return 1
    print('PASS: database projections match schema.sql' if args.check else 'Generated 5 database projections');return 0
if __name__=='__main__':sys.exit(main())
