const assert = require('node:assert/strict');
const fs = require('node:fs/promises');
const path = require('node:path');
const {test} = require('node:test');
const {chromium, webkit} = require('playwright');
const assets=path.resolve(__dirname,'../assets');
const {I18N}=require('./strings.cjs').strings();
for(const engine of (process.env.BROWSER?[process.env.BROWSER]:['chromium','webkit'])) for(const lang of ['en','zh','zh-TW','ja','de']) {
 test(`${engine} ${lang}: cooldown defaults, persistence, validation and inheritance`,async(t)=>{
  const browser=await(engine==='webkit'?webkit.launch():chromium.launch({channel:'chromium'}));t.after(()=>browser.close());
  const page=await browser.newPage({viewport:{width:420,height:850},reducedMotion:'reduce'});page.setDefaultTimeout(5000);
  const errors=[];page.on('pageerror',e=>errors.push(e.message));
  const tr=s=>I18N[lang]?.[s]||s;
  let raw={}, reject=false;const posts=[], reads=[];
  const base={icon:"generic",chat:"https://example.invalid/v1",models:[{id:"m",name:"M",on:true}],agents:[],fallback:[],headers:{},balanceToken:{takes:false,set:false},proxy:""};
  const relay={...base,id:"relay",name:"Relay",key:{set:true,masked:"sk-…a"},keyList:[{id:"a",name:"A",masked:"sk-…a",on:true,active:true},{id:"b",name:"B",masked:"sk-…b",on:true}]};
  const codex={...base,id:"codex",name:"Codex",keyList:[],account:{agent:"codex",agentName:"Codex",user:"me@example.com",plan:"PLUS",logins:[{user:"me@example.com",plan:"PLUS",active:true,on:true}]}};
  const defaults={rateStart:2,rateCap:64,slowStart:1,slowThreshold:2,slowCap:10,failureCap:10,quotaRest:30,quotaResetAt:'',failRest:true,limitRest:true,models:{},source:{}};
  await page.route('**/*',async route=>{
   const url=new URL(route.request().url()), json=data=>route.fulfill({json:data});
   if(url.pathname==='/boot.js')return route.fulfill({contentType:'text/javascript',body:`window.bootPrefs={lang:${JSON.stringify(lang)},theme:'light',web:true};`});
   if(url.pathname==='/wails/runtime.js')return route.fulfill({contentType:'text/javascript',body:'export const Window={};'});
   if(url.pathname==='/api/state')return json({agents:[],profiles:[],settings:{lang,theme:'light'}});
   if(url.pathname==='/api/providers')return json({providers:[relay,codex],presets:[],excluded:[],gateway:{running:true,window:true}});
   if(url.pathname==='/api/groups')return json({groups:[]});
   if(url.pathname==='/api/cooldown'){
    reads.push({id:url.searchParams.get('id'),ref:url.searchParams.get('ref')});
    if(route.request().method()==='POST'){
     posts.push(route.request().postDataJSON());
     if(reject)return route.fulfill({status:400,json:{error:'fixture save failed'}});
     raw=route.request().postDataJSON();
    }
    return json({raw,legacy:{},inherited:defaults,effective:{...defaults,...raw,source:Object.fromEntries(Object.keys(raw).map(k=>[k,'global']))},rests:[]});
   }
   if(url.pathname.startsWith('/api/'))return json({});
   const file=path.join(assets,url.pathname==='/'?'index.html':url.pathname);
   await route.fulfill({body:await fs.readFile(file),contentType:{'.html':'text/html','.js':'text/javascript','.css':'text/css','.svg':'image/svg+xml'}[path.extname(file)]});
  });
  await page.goto('http://magpie.test/?view=settings&tab=network');
  await page.locator('#cooldownDefaults').click();
  const editor=page.locator('.cooldown-editor');await editor.waitFor();
  const header=await editor.locator(".ehead").boundingBox();assert.ok(header.y>=0,"dialog title is outside the viewport");
  const footer=await editor.locator(".bar").boundingBox();assert.ok(footer.y+footer.height<=851,"save controls are outside the viewport");
  assert.equal(await editor.locator('[data-cooldown=rateStart]').getAttribute('placeholder'),'2');
  assert.equal(await editor.locator('[data-cooldown=rateCap]').getAttribute('placeholder'),'64');
  assert.equal(await editor.locator('[data-cooldown=quotaRest]').getAttribute('placeholder'),'30');
  await editor.locator('[data-cooldown=rateStart]').fill('4');
  await editor.locator('[data-cooldown=rateCap]').fill('32');
  assert.match(await editor.locator('.cooldown-ladder').textContent(),/4 → 8 → 16 → 32/);
  const save=editor.getByRole('button',{name:tr('Save'),exact:true});
  reject=true;await save.click();await editor.getByRole('alert').filter({hasText:'fixture save failed'}).waitFor();
  assert.equal(await editor.locator('[data-cooldown=rateCap]').inputValue(),'32');
  reject=false;await save.click();
  await page.waitForFunction(()=>document.querySelector('.cooldown-editor [data-cooldown=rateCap]')?.value==='32' && !document.querySelector('.cooldown-editor button.primary')?.disabled);
  assert.equal(posts.at(-1).rateCap,32);
  await editor.locator('[data-cooldown=rateStart]').fill('40');await editor.getByRole('button',{name:tr('Save'),exact:true}).click();
  await editor.getByRole('alert').filter({hasText:tr('The first rest cannot exceed its cap.')}).waitFor();
  assert.equal(posts.length,2);
  await editor.locator('[data-cooldown=rateStart]').fill('');await editor.locator('[data-cooldown=rateCap]').fill('');
  assert.match(await editor.locator('.cooldown-ladder').textContent(),/2 → 4 → 8 → 16 → 32 → 64/);
  await editor.getByRole('button',{name:tr('Save'),exact:true}).click();
  await page.waitForFunction(()=>document.querySelector('.cooldown-editor [data-cooldown=rateCap]')?.value==='');
  assert.equal(posts.at(-1).rateCap,undefined);
  assert.ok(await editor.evaluate(el=>el.scrollWidth<=el.clientWidth+1),'editor overflows on narrow screen');
  await editor.getByRole('button',{name:tr('Add model cooldown'),exact:true}).click();
  await editor.getByRole('textbox',{name:tr('Exact model ID'),exact:true}).fill('m');
  await editor.getByRole('spinbutton',{name:tr('Model cap (0 disables; blank inherits)'),exact:true}).fill('60');
  await editor.getByRole('button',{name:tr('Save'),exact:true}).click();
  await page.waitForFunction(()=>document.querySelector('.cooldown-model input[type=number]')?.value==='60');
  assert.equal(posts.at(-1).models.m,60);
  await editor.getByRole('button',{name:tr('Cancel'),exact:true}).click();
  await page.goto('http://magpie.test/?view=providers');
  await page.locator('.row.provider[data-id=relay] .cooldown-open').click();
  await editor.waitFor();assert.deepEqual(reads.at(-1),{id:'relay',ref:''});
  await editor.getByRole('button',{name:tr('Cancel'),exact:true}).click();
  await page.locator('.row.provider[data-id=relay] .who').click();
  await page.locator('.acc[data-account-id=a] .cooldown-open').click();
  await editor.waitFor();assert.deepEqual(reads.at(-1),{id:'relay',ref:'a'});
  await page.evaluate(()=>renderProviders());assert.equal(await editor.count(),1,'provider refresh closed settings');
  await editor.getByRole('button',{name:tr('Cancel'),exact:true}).click();
  await page.locator('.row.provider[data-id=codex] .who').click();
  await page.locator('.acc[data-account-id="me@example.com"] .cooldown-open').click();
  await editor.waitFor();assert.deepEqual(reads.at(-1),{id:'codex',ref:'me@example.com'});
  await editor.locator('[data-cooldown=rateCap]').fill('16');
  await editor.getByRole('button',{name:tr('Save'),exact:true}).click();
  await page.waitForFunction(()=>document.querySelector('.cooldown-editor [data-cooldown=rateCap]')?.value==='16');
  assert.equal(posts.at(-1).rateCap,16);
  if(process.env.ARTIFACT_DIR){await fs.mkdir(process.env.ARTIFACT_DIR,{recursive:true});await editor.locator('.ebody').evaluate(el=>{el.scrollTop=0;});await page.screenshot({path:path.join(process.env.ARTIFACT_DIR,`${engine}-${lang}-cooldown.png`)});}
  assert.deepEqual(errors,[]);
 });
}
