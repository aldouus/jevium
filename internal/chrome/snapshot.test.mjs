import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {runInNewContext} from 'node:vm';

const source=readFileSync(new URL('./snapshot.js',import.meta.url),'utf8');
function control(tagName,label,value,extra={}) {
  return {tagName,type:'text',value,readOnly:false,disabled:false,isConnected:true,
    labels:[],childNodes:[],parentElement:{innerText:''},
    getAttribute(name){return name==='aria-label'?label:null},
    closest(){return null},checkVisibility(){return true},matches(){return false},
    getBoundingClientRect(){return {x:0,y:0,width:100,height:40}},...extra};
}
function snapshot(elements) {
  const document={body:{},title:'Fixture',documentElement:{scrollHeight:800},
    querySelectorAll(){return elements},getElementById(){return null},
    createTreeWalker(){return {nextNode(){return null}}},createRange(){return {}}};
  return runInNewContext(source,{document,window:{},performance:{timeOrigin:1},
    location:{href:'https://example.test'},scrollX:0,scrollY:0,innerWidth:400,innerHeight:800,
    NodeFilter:{SHOW_TEXT:4}});
}
test('readonly values remain observable without a fill operation',()=>{
  const p=snapshot([control('INPUT','Account','123',{readOnly:true})]);
  assert.equal(p.controls.length,1);
  assert.equal(p.controls[0].value,'123');
  assert.equal(p.actions.some(a=>a.kind==='fill'),false);
});
test('selected value survives when no alternative option is actionable',()=>{
  const p=snapshot([control('SELECT','Country','ca',{options:[
    {label:'Canada',value:'ca',selected:true,disabled:false,closest(){return null}},
    {label:'France',value:'fr',selected:false,disabled:true,closest(){return null}},
  ]})]);
  assert.equal(p.controls.length,1);
  assert.equal(p.controls[0].label,'Country');
  assert.equal(p.controls[0].value,'ca');
  assert.equal(p.actions.some(a=>a.kind==='select'),false);
});
test('hidden controls and non-value buttons do not synthesize empty fields',()=>{
  const p=snapshot([
    control('INPUT','Hidden','secret',{checkVisibility(){return false}}),
    control('BUTTON','Submit',''),
  ]);
  assert.equal(p.controls.length,1);
  assert.equal(p.controls[0].label,'Submit');
  assert.equal('value' in p.controls[0],false);
});
