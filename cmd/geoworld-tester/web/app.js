const $=id=>document.getElementById(id);

let active=null;
let layer='final';
let poll=null;
let busy=false;
let serviceOnline=true;
let mapLoadToken=0;
let mapObjectURL=null;
let renderedSeed=null;
let modelNameTouched=false;

const stageNames={
  phase1:'地壳与大尺度地形',
  climate:'气候',
  hydrology:'水系',
  erosion:'侵蚀',
  landform:'地貌',
  geomorph:'地形结构',
  ecology:'生态',
  mcprofile:'Minecraft 配置',
  terrain:'连续地形',
  reconcile:'海陆校正',
  waterbed:'河床与湖床',
  worldsample:'最终地面与水面',
  runtimeprofile:'最终群系配置'
};

class ApiError extends Error{
  constructor(message,{network=false,status=0}={}){
    super(message);
    this.network=network;
    this.status=status;
  }
}

function backendMessage(){
  return '无法连接 GeoWorld Tester 后端。请确认启动 Tester 的命令窗口仍在运行；当前地图没有更新。';
}

async function api(url,opts={}){
  let r;
  try{
    r=await fetch(url,{headers:{'Content-Type':'application/json'},...opts});
  }catch(err){
    updateBackend(false);
    throw new ApiError(backendMessage(),{network:true});
  }
  updateBackend(true);
  const data=await r.json().catch(()=>({}));
  if(!r.ok)throw new ApiError(data.error||('HTTP '+r.status),{status:r.status});
  return data;
}

function updateBackend(online){
  const wasOnline=serviceOnline;
  serviceOnline=online;
  const el=$('backendStatus');
  el.textContent=online?'后端在线':'后端断开';
  el.classList.toggle('online',online);
  el.classList.toggle('offline',!online);

  if(!online){
    $('topStatus').textContent='服务断开';
    if(active||renderedSeed!==null){
      setMapNotice('后端已断开；当前仍显示上一次成功加载的地图。','error');
    }
  }else if(!wasOnline){
    const staleSeed=renderedSeed;
    active=null;
    clearSample();
    renderEmptyStats();
    $('topStatus').textContent='需重新加载';
    $('progressText').textContent='后端已恢复，请重新生成世界或重新打开已保存模型';
    if(staleSeed!==null){
      setMapNotice(`后端已恢复；当前仍是 Seed ${staleSeed} 的旧图，请重新生成或重新打开模型。`,'warn');
    }else{
      setMapNotice('');
    }
    void refreshModels();
  }
  updateActionState();
}

function setBusy(v){
  busy=v;
  updateActionState();
}

function updateActionState(){
  $('generate').disabled=busy||!serviceOnline;
  $('save').disabled=busy||!serviceOnline||!active||active.source!=='generated';
  $('load').disabled=busy||!serviceOnline||!$('models').value;
}

function setMapNotice(text='',kind=''){
  const el=$('mapNotice');
  if(!text){
    el.hidden=true;
    el.textContent='';
    el.className='map-notice';
    return;
  }
  el.hidden=false;
  el.textContent=text;
  el.className='map-notice '+kind;
}

function stageLabel(stage){
  return stageNames[stage]||stage||'准备';
}

async function refreshStatus(){
  try{
    const s=await api('/api/status');
    if(s.generating){
      const pct=s.total?Math.round(s.completed/s.total*100):0;
      $('progressBar').style.width=pct+'%';
      $('progressText').textContent=`正在生成：${stageLabel(s.stage)} · ${s.completed}/${s.total} · ${((s.elapsedMs||0)/1000).toFixed(1)}s`;
      $('topStatus').textContent='生成中';
    }
  }catch(e){
    if(e.network){
      $('topStatus').textContent='服务断开';
      if(active)setMapNotice('后端已断开；当前仍显示上一次成功加载的地图。','error');
    }
  }
}

function updateRangePreview(){
  const w=Math.max(1,Number($('width').value)||0);
  const h=Math.max(1,Number($('height').value)||0);
  const cell=Number($('cellSize').value)||0;
  const sx=Math.max(0,w-1)*cell;
  const sz=Math.max(0,h-1)*cell;
  $('rangePreview').textContent=`当前覆盖约 ${Math.round(sx).toLocaleString()} × ${Math.round(sz).toLocaleString()} 方块；间距越小，细节越密，但同样网格覆盖范围越小。`;
}

function updateContinentLabels(){
  $('fragmentationValue').textContent=Number($('fragmentation').value).toFixed(2);
  $('separationValue').textContent=Number($('separation').value).toFixed(2);
}

['width','height','cellSize'].forEach(id=>$(id).addEventListener('input',updateRangePreview));
['fragmentation','separation'].forEach(id=>$(id).addEventListener('input',updateContinentLabels));
$('modelName').addEventListener('input',()=>{modelNameTouched=true});
$('randomSeed').addEventListener('click',()=>{
  const values=new Uint32Array(1);
  crypto.getRandomValues(values);
  $('seed').value=String((values[0]&0x7fffffff)||1);
});

$('generate').onclick=async()=>{
  const previous=active;
  setBusy(true);
  $('progressBar').style.width='0%';
  $('progressText').textContent='开始生成…';
  $('modelMessage').textContent='';
  if(previous){
    setMapNotice(`正在生成新世界；当前地图暂时仍是 Seed ${previous.seed} 的上一次成功结果。`,'warn');
  }else{
    setMapNotice('正在生成世界…','warn');
  }
  poll=setInterval(refreshStatus,250);
  try{
    const next=await api('/api/generate',{method:'POST',body:JSON.stringify({
      seed:Number($('seed').value),
      width:Number($('width').value),
      height:Number($('height').value),
      cellSize:Number($('cellSize').value),
      plateCount:Number($('plates').value),
      oceanFraction:Number($('ocean').value),
      continentFragmentation:Number($('fragmentation').value),
      continentSeparation:Number($('separation').value)
    })});
    active=next;
    $('progressBar').style.width='100%';
    $('progressText').textContent=`生成完成 · ${(active.durationMs/1000).toFixed(2)}s`;
    $('topStatus').textContent='已生成';
    if(!modelNameTouched||!$('modelName').value.trim()){
      $('modelName').value='seed-'+active.seed;
      modelNameTouched=false;
    }
    clearSample();
    renderStats();
    await loadMap();
    await refreshModels();
  }catch(e){
    $('progressText').innerHTML='<span class="error">'+escapeHtml(e.message)+'</span>';
    $('topStatus').textContent=e.network?'服务断开':'生成失败';
    if(previous){
      setMapNotice(`生成失败；当前地图仍是 Seed ${previous.seed} 的上一次成功结果。`,'error');
    }else{
      setMapNotice('生成失败，没有可显示的新地图。','error');
    }
  }finally{
    clearInterval(poll);
    poll=null;
    setBusy(false);
  }
};

document.querySelectorAll('.layer').forEach(btn=>btn.onclick=()=>{
  document.querySelectorAll('.layer').forEach(b=>b.classList.remove('active'));
  btn.classList.add('active');
  layer=btn.dataset.layer;
  if(active)void loadMap();
});

async function loadMap(){
  if(!active)return;
  const token=++mapLoadToken;
  const targetSeed=active.seed;
  const oldSeed=renderedSeed;
  setMapNotice(`正在渲染 ${layerName(layer)}…`,'warn');
  let r;
  try{
    r=await fetch(`/api/layer?name=${encodeURIComponent(layer)}&width=720&height=450&t=${Date.now()}`);
  }catch(err){
    updateBackend(false);
    setMapNotice(oldSeed===null?'地图加载失败：后端已断开。':`地图加载失败；当前仍显示 Seed ${oldSeed} 的旧地图。`,'error');
    return;
  }
  updateBackend(true);
  if(!r.ok){
    const data=await r.json().catch(()=>({}));
    setMapNotice('地图加载失败：'+(data.error||('HTTP '+r.status)),'error');
    return;
  }
  const blob=await r.blob();
  if(token!==mapLoadToken)return;
  const nextURL=URL.createObjectURL(blob);
  if(mapObjectURL)URL.revokeObjectURL(mapObjectURL);
  mapObjectURL=nextURL;
  $('map').src=nextURL;
  $('mapWrap').classList.remove('empty');
  renderedSeed=targetSeed;
  setMapNotice('');
}

function layerName(v){
  return ({final:'最终地形',water:'水体',biome:'群系',profile:'配置置信度'})[v]||v;
}

$('map').onclick=async e=>{
  if(!active)return;
  const rect=$('map').getBoundingClientRect();
  const fx=(e.clientX-rect.left)/rect.width;
  const fz=(e.clientY-rect.top)/rect.height;
  const g=active.geometry;
  const x=g.OriginX+fx*g.SpanXBlocks;
  const z=g.OriginZ+fz*g.SpanZBlocks;
  const c=$('cursor');
  c.style.display='block';
  c.style.left=(e.clientX-$('mapWrap').getBoundingClientRect().left-6)+'px';
  c.style.top=(e.clientY-$('mapWrap').getBoundingClientRect().top-6)+'px';
  try{
    renderSample(await api(`/api/sample?x=${x}&z=${z}`));
  }catch(err){
    $('sampleEmpty').hidden=true;
    $('sample').hidden=false;
    $('sample').innerHTML='<div class="error-card">'+escapeHtml(err.message)+'</div>';
  }
};

$('save').onclick=async()=>{
  $('modelMessage').textContent='';
  try{
    const r=await api('/api/save',{method:'POST',body:JSON.stringify({name:$('modelName').value})});
    $('modelMessage').innerHTML='<span class="ok">已保存：'+escapeHtml(r.name)+'.geoworld</span>';
    await refreshModels(r.name);
  }catch(e){
    $('modelMessage').innerHTML='<span class="error">'+escapeHtml(e.message)+'</span>';
  }
};

$('load').onclick=async()=>{
  const name=$('models').value;
  if(!name)return;
  setBusy(true);
  $('modelMessage').textContent='';
  try{
    active=await api('/api/load',{method:'POST',body:JSON.stringify({name})});
    $('seed').value=active.seed;
    $('width').value=active.geometry.Width;
    $('height').value=active.geometry.Height;
    const cell=active.geometry.Width>1?active.geometry.SpanXBlocks/(active.geometry.Width-1):0;
    if(cell===64||cell===128)$('cellSize').value=String(cell);
    $('plates').value=active.plateCount;
    $('ocean').value=active.oceanFraction;
    $('fragmentation').value=active.continentFragmentation||0.35;
    $('separation').value=active.continentSeparation||0.65;
    updateRangePreview();
    updateContinentLabels();
    $('topStatus').textContent='已重开';
    $('progressText').textContent=`模型重开完成 · ${active.durationMs}ms`;
    $('modelName').value=name;
    modelNameTouched=true;
    $('modelMessage').innerHTML='<span class="warn">当前是已保存模型的只读重开结果；如需再次保存，请重新生成世界。</span>';
    clearSample();
    renderStats();
    await loadMap();
  }catch(e){
    $('modelMessage').innerHTML='<span class="error">'+escapeHtml(e.message)+'</span>';
  }finally{
    setBusy(false);
  }
};

async function refreshModels(selectName=''){
  const previous=selectName||$('models').value;
  try{
    const list=await api('/api/models');
    $('models').innerHTML=list.length
      ?list.map(m=>`<option value="${escapeAttr(m.name)}">${escapeHtml(m.name)} · Seed ${m.seed} · ${formatBytes(m.fileBytes)}</option>`).join('')
      :'<option value="">暂无</option>';
    if(previous&&list.some(m=>m.name===previous))$('models').value=previous;
  }catch(e){
    $('models').innerHTML='<option value="">读取失败</option>';
    if(e.network)$('modelMessage').innerHTML='<span class="error">'+escapeHtml(e.message)+'</span>';
  }
  updateActionState();
}

function renderEmptyStats(){
  $('stats').outerHTML='<div id="stats" class="kv"><span>Seed</span><b>—</b></div>';
}

function renderStats(){
  if(!active)return;
  const g=active.geometry;
  const cell=g.Width>1?g.SpanXBlocks/(g.Width-1):0;
  $('stats').outerHTML=`<div id="stats">
    ${kv('来源',active.source==='generated'?'刚生成':'模型重开')}
    ${kv('Seed',active.seed)}
    ${kv('网格',g.Width+' × '+g.Height)}
    ${kv('采样间距',Math.round(cell)+' 方块 / 点')}
    ${kv('世界范围',Math.round(g.SpanXBlocks)+' × '+Math.round(g.SpanZBlocks)+' 方块')}
    ${kv('板块数',active.plateCount)}
    ${kv('洋壳比例',Number(active.oceanFraction).toFixed(2))}
    ${kv('大陆破碎度',Number(active.continentFragmentation).toFixed(2))}
    ${kv('大陆分离度',Number(active.continentSeparation).toFixed(2))}
    ${kv('主要陆块',active.majorLandComponents)}
    ${kv('全部陆块',active.landComponents)}
    ${kv('最大陆块占比',(Number(active.largestLandShare)*100).toFixed(1)+'%')}
    ${kv('河段',active.riverSegments)}
    ${kv('湖泊节点',active.lakeNodes)}
  </div>`;
}

function clearSample(){
  $('cursor').style.display='none';
  $('sampleEmpty').hidden=false;
  $('sample').hidden=true;
  $('sample').innerHTML='';
}

function renderSample(s){
  $('sampleEmpty').hidden=true;
  $('sample').hidden=false;
  const water=s.waterSurfaceDefined
    ?`${s.waterKind} · 水面 ${f(s.waterSurfaceElevation)} · 深度 ${f(s.waterDepth)}`
    :s.waterKind;
  $('sample').innerHTML=`
    <div class="sample-section"><h3>最终地面 / 水体</h3>
      ${kv('坐标',Math.round(s.x)+', '+Math.round(s.z))}
      ${kv('地面高度',f(s.groundHeight))}
      ${kv('水体',water)}
      ${kv('海洋判定权重',f(s.oceanEligibility))}
      ${kv('河流权重',f(s.riverWeight))}
    </div>
    <div class="sample-section"><h3>生态 / Minecraft</h3>
      ${kv('生态区',s.ecoRegion)}
      ${kv('生物群系',s.biomeFamily)}
      ${kv('地形类型',s.terrainVariant)}
      ${kv('地表方案',s.surfaceProfile)}
      ${kv('植被方案',s.vegetationProfile)}
      ${kv('配置置信度',f(s.profileConfidence))}
    </div>
    <div class="sample-section"><h3>一致性检查</h3>
      ${kv('来源节点',s.sourceNode)}
      ${kv('来源是否陆地',s.sourceLand?'是':'否')}
      ${kv('语义回退',s.semanticFallback?'<span class="error">是</span>':'<span class="ok">否</span>')}
      ${kv('语义冲突',s.semanticMismatch?'<span class="error">是</span>':'<span class="ok">否</span>')}
    </div>`;
}

function kv(k,v){return `<div class="kv"><span>${k}</span><b>${v}</b></div>`}
function f(v){return Number(v).toFixed(6)}
function formatBytes(n){return n<1048576?(n/1024).toFixed(1)+' KB':(n/1048576).toFixed(2)+' MB'}
function escapeHtml(s){return String(s).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]))}
function escapeAttr(s){return escapeHtml(s)}

window.addEventListener('beforeunload',()=>{if(mapObjectURL)URL.revokeObjectURL(mapObjectURL)});

updateRangePreview();
updateContinentLabels();
updateActionState();
refreshModels();
refreshStatus();
setInterval(()=>{if(!poll)void refreshStatus()},2000);
