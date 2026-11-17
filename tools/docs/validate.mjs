import { Parser } from '@dbml/core';
import {JSDOM} from 'jsdom';
import fs from 'node:fs';
import {fileURLToPath} from 'node:url';
const dom=new JSDOM('<!DOCTYPE html><html><body></body></html>');
globalThis.window=dom.window;globalThis.document=dom.window.document;
const {default: mermaid}=await import('mermaid');
mermaid.initialize({startOnLoad:false,securityLevel:'strict'});
const root=fileURLToPath(new URL('../../',import.meta.url));
try {
 const db=new Parser().parse(fs.readFileSync(root+'/docs/database/schema.dbml','utf8'),'dbml');
 const inv=JSON.parse(fs.readFileSync(root+'/docs/database/inventory.json','utf8'));
 const tables=db.schemas.flatMap(s=>s.tables);
 const refs=db.schemas.flatMap(s=>s.refs);
 if(tables.length!==inv.table_count || tables.reduce((n,t)=>n+t.fields.length,0)!==inv.column_count || refs.length!==inv.foreign_key_count) throw new Error('DBML counts differ from AST inventory');
 console.log('DBML PASS:',tables.length,'tables,',inv.column_count,'columns,',refs.length,'FK');
} catch(e){console.error(e.message,JSON.stringify(e.diags));process.exitCode=1;}
for(const p of ['docs/database/er-diagram.mmd','docs/architecture/diagrams/system.mmd','docs/architecture/diagrams/deployment.mmd','docs/architecture/diagrams/sequence.mmd']){
 try{await mermaid.parse(fs.readFileSync(root+'/'+p,'utf8'));console.log('MERMAID PASS:',p)}catch(e){console.error('FAIL',p,e.message);process.exitCode=1;}
}
