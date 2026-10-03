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
let exportNameTouched=false;
let lastExport=null;
let mapZoom=1;
let mapPanX=0;
let mapPanY=0;
let mapDragging=false;
let mapDragMoved=false;
let mapDragStartX=0;
let mapDragStartY=0;
let mapPanStartX=0;
let mapPanStartY=0;
let suppressMapClick=false;
let lastFocusPoint=null;

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


const continentPresets={
  default:{
    ocean:.68,fragmentation:.35,separation:.65,
    help:'64 Seed 基线（Seed 1–64，320×200 / 128）：平均主要陆块 2.61、全部陆块 6.14、最大陆块占比 69.9%。'
  },
  super:{
    ocean:.50,fragmentation:.05,separation:.05,
    help:'64 Seed 基线：50/64 达到“主要陆块=1 且最大陆块≥85%”；平均主要陆块 1.23、最大陆块占比 96.8%。'
  },
  multi:{
    ocean:.68,fragmentation:.30,separation:.90,
    help:'64 Seed 基线：54/64 达到“主要陆块≥2 且最大陆块≤80%”；平均主要陆块 3.48、最大陆块占比 57.0%。'
  },
  broken:{
    ocean:.68,fragmentation:.75,separation:.75,
    help:'64 Seed 基线：平均主要陆块 2.69、全部陆块 8.86；强调海峡、半岛和次级陆块，不保证没有大型大陆。'
  },
  archipelago:{
    ocean:.90,fragmentation:.95,separation:.95,
    help:'64 Seed 基线：62/64 达到“全部陆块≥10 且最大陆块≤65%”；平均主要陆块 6.73、全部陆块 15.61、最大陆块占比 33.4%。'
  }
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
    clearExportResult();
    renderEmptyStats();
    renderEmptyAnomalies();
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

function clearExportResult({message=true}={}){
  lastExport=null;
  $('openExport').hidden=true;
  $('exportBounds').hidden=true;
  $('spawnMarker').hidden=true;
  if(message)$('exportMessage').textContent='生成或重开模型后即可导出。';
  updateActionState();
}

function clampMapZoom(v){
  return Math.max(1,Math.min(8,v));
}

function applyMapTransform(){
  const img=$('map');
  const wrap=$('mapWrap');
  img.style.transform=`translate(${mapPanX}px,${mapPanY}px) scale(${mapZoom})`;
  wrap.classList.toggle('zoomed',mapZoom>1.001);
  $('mapZoomLabel').textContent=mapZoom.toFixed(1)+'×';
  updateActionState();
  requestAnimationFrame(()=>{
    renderExportOverlay();
    renderCursorOverlay();
  });
}

function resetMapView(){
  mapZoom=1;
  mapPanX=0;
  mapPanY=0;
  applyMapTransform();
}

function zoomMapAt(clientX,clientY,nextZoom){
  nextZoom=clampMapZoom(nextZoom);
  if(Math.abs(nextZoom-mapZoom)<1e-6)return;
  if(nextZoom<=1.001){
    resetMapView();
    return;
  }
  const img=$('map');
  const oldRect=img.getBoundingClientRect();
  if(oldRect.width<=0||oldRect.height<=0)return;
  const fx=(clientX-oldRect.left)/oldRect.width;
  const fz=(clientY-oldRect.top)/oldRect.height;

  mapZoom=nextZoom;
  img.style.transform=`translate(${mapPanX}px,${mapPanY}px) scale(${mapZoom})`;
  const nextRect=img.getBoundingClientRect();
  mapPanX+=clientX-(nextRect.left+fx*nextRect.width);
  mapPanY+=clientY-(nextRect.top+fz*nextRect.height);
  applyMapTransform();
}

function zoomMapCentered(factor){
  const wrap=$('mapWrap').getBoundingClientRect();
  zoomMapAt(
    wrap.left+wrap.width/2,
    wrap.top+wrap.height/2,
    mapZoom*factor
  );
}

function renderCursorOverlay(){
  const c=$('cursor');
  if(!active||!lastFocusPoint||$('mapWrap').classList.contains('empty')){
    c.style.display='none';
    return;
  }
  const g=active.geometry;
  const fx=Math.max(0,Math.min(1,(lastFocusPoint.x-g.OriginX)/g.SpanXBlocks));
  const fz=Math.max(0,Math.min(1,(lastFocusPoint.z-g.OriginZ)/g.SpanZBlocks));
  const imgRect=$('map').getBoundingClientRect();
  const wrapRect=$('mapWrap').getBoundingClientRect();
  if(imgRect.width<=0||imgRect.height<=0){
    c.style.display='none';
    return;
  }
  c.style.display='block';
  c.style.left=(imgRect.left-wrapRect.left+fx*imgRect.width-6)+'px';
  c.style.top=(imgRect.top-wrapRect.top+fz*imgRect.height-6)+'px';
}

function renderExportOverlay(){
  const bounds=$('exportBounds');
  const spawn=$('spawnMarker');
  if(!active||!lastExport||$('mapWrap').classList.contains('empty')){
    bounds.hidden=true;
    spawn.hidden=true;
    return;
  }

  const g=active.geometry;
  const worldMinX=(lastExport.centerChunkX-lastExport.radiusChunks)*16;
  const worldMaxX=(lastExport.centerChunkX+lastExport.radiusChunks+1)*16;
  const worldMinZ=(lastExport.centerChunkZ-lastExport.radiusChunks)*16;
  const worldMaxZ=(lastExport.centerChunkZ+lastExport.radiusChunks+1)*16;
  const modelMinX=g.OriginX;
  const modelMaxX=g.OriginX+g.SpanXBlocks;
  const modelMinZ=g.OriginZ;
  const modelMaxZ=g.OriginZ+g.SpanZBlocks;

  if(
    worldMaxX<=modelMinX||worldMinX>=modelMaxX||
    worldMaxZ<=modelMinZ||worldMinZ>=modelMaxZ
  ){
    bounds.hidden=true;
    spawn.hidden=true;
    return;
  }

  const fx0=Math.max(0,Math.min(1,(worldMinX-modelMinX)/g.SpanXBlocks));
  const fx1=Math.max(0,Math.min(1,(worldMaxX-modelMinX)/g.SpanXBlocks));
  const fz0=Math.max(0,Math.min(1,(worldMinZ-modelMinZ)/g.SpanZBlocks));
  const fz1=Math.max(0,Math.min(1,(worldMaxZ-modelMinZ)/g.SpanZBlocks));

  const imgRect=$('map').getBoundingClientRect();
  const wrapRect=$('mapWrap').getBoundingClientRect();
  if(imgRect.width<=0||imgRect.height<=0){
    bounds.hidden=true;
    spawn.hidden=true;
    return;
  }

  const baseLeft=imgRect.left-wrapRect.left;
  const baseTop=imgRect.top-wrapRect.top;
  bounds.hidden=false;
  bounds.style.left=(baseLeft+fx0*imgRect.width)+'px';
  bounds.style.top=(baseTop+fz0*imgRect.height)+'px';
  bounds.style.width=Math.max(2,(fx1-fx0)*imgRect.width)+'px';
  bounds.style.height=Math.max(2,(fz1-fz0)*imgRect.height)+'px';
  bounds.title='Minecraft 预生成范围：radius '+lastExport.radiusChunks+' Chunk';

  const sfx=Math.max(0,Math.min(1,(lastExport.spawnX-modelMinX)/g.SpanXBlocks));
  const sfz=Math.max(0,Math.min(1,(lastExport.spawnZ-modelMinZ)/g.SpanZBlocks));
  spawn.hidden=false;
  spawn.style.left=(baseLeft+sfx*imgRect.width)+'px';
  spawn.style.top=(baseTop+sfz*imgRect.height)+'px';
  spawn.title='Minecraft 出生点：'+lastExport.spawnX+', '+lastExport.spawnY+', '+lastExport.spawnZ;
}

function updateActionState(){
  $('generate').disabled=busy||!serviceOnline;
  $('save').disabled=busy||!serviceOnline||!active||active.source!=='generated';
  $('load').disabled=busy||!serviceOnline||!$('models').value;
  $('exportMinecraft').disabled=busy||!serviceOnline||!active;
  $('openExport').disabled=busy||!serviceOnline||!lastExport;
  const mapDisabled=busy||!serviceOnline||!active||$('mapWrap').classList.contains('empty');
  $('mapZoomIn').disabled=mapDisabled||mapZoom>=7.999;
  $('mapZoomOut').disabled=mapDisabled||mapZoom<=1.001;
  $('mapZoomReset').disabled=mapDisabled||(mapZoom<=1.001&&Math.abs(mapPanX)<0.5&&Math.abs(mapPanY)<0.5);
  const debugDisabled=busy||!serviceOnline||!active||!active.intermediateDiagnostics;
  document.querySelectorAll('.debug-layer').forEach(btn=>{btn.disabled=debugDisabled});
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
    }else if(s.exporting){
      $('topStatus').textContent='导出中';
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


function nearlyEqual(a,b){
  return Math.abs(Number(a)-Number(b))<1e-9;
}

function matchingContinentPreset(){
  const ocean=Number($('ocean').value);
  const fragmentation=Number($('fragmentation').value);
  const separation=Number($('separation').value);
  for(const [name,p] of Object.entries(continentPresets)){
    if(nearlyEqual(ocean,p.ocean)&&
       nearlyEqual(fragmentation,p.fragmentation)&&
       nearlyEqual(separation,p.separation)){
      return name;
    }
  }
  return 'custom';
}

function updatePresetHelp(){
  const name=$('continentPreset').value;
  const p=continentPresets[name];
  if(!p){
    $('presetHelp').textContent='自定义：当前三个参数不属于内置预设；生成与保存仍直接使用输入框中的实际数值。';
    return;
  }
  $('presetHelp').textContent=
    `洋壳 ${p.ocean.toFixed(2)} · 破碎 ${p.fragmentation.toFixed(2)} · 分离 ${p.separation.toFixed(2)}。 ${p.help}`;
}

function syncContinentPreset(){
  $('continentPreset').value=matchingContinentPreset();
  updatePresetHelp();
}

function applyContinentPreset(name){
  const p=continentPresets[name];
  if(!p){
    updatePresetHelp();
    return;
  }
  $('ocean').value=p.ocean.toFixed(2);
  $('fragmentation').value=p.fragmentation.toFixed(2);
  $('separation').value=p.separation.toFixed(2);
  updateContinentLabels();
  updatePresetHelp();
}

function updateContinentLabels(){
  $('fragmentationValue').textContent=Number($('fragmentation').value).toFixed(2);
  $('separationValue').textContent=Number($('separation').value).toFixed(2);
}

['width','height','cellSize'].forEach(id=>$(id).addEventListener('input',updateRangePreview));
['ocean','fragmentation','separation'].forEach(id=>$(id).addEventListener('input',()=>{
  updateContinentLabels();
  syncContinentPreset();
}));
$('continentPreset').addEventListener('change',()=>applyContinentPreset($('continentPreset').value));
$('modelName').addEventListener('input',()=>{modelNameTouched=true});
$('exportName').addEventListener('input',()=>{exportNameTouched=true});
$('randomSeed').addEventListener('click',()=>{
  const values=new Uint32Array(1);
  crypto.getRandomValues(values);
  $('seed').value=String((values[0]&0x7fffffff)||1);
});

$('generate').onclick=async()=>{
  const previous=active;
  setBusy(true);
  clearExportResult();
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
    resetMapView();
    $('progressBar').style.width='100%';
    $('progressText').textContent=`生成完成 · ${(active.durationMs/1000).toFixed(2)}s`;
    $('topStatus').textContent='已生成';
    if(!modelNameTouched||!$('modelName').value.trim()){
      $('modelName').value='seed-'+active.seed;
      modelNameTouched=false;
    }
    if(!exportNameTouched||!$('exportName').value.trim()){
      $('exportName').value='GeoWorld '+active.seed+' QA';
      exportNameTouched=false;
    }
    clearSample();
    renderStats();
    await refreshAnomalies();
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

function selectLayer(next,{load=true}={}){
  layer=next;
  document.querySelectorAll('.layer').forEach(b=>b.classList.toggle('active',b.dataset.layer===next));
  updateLayerHelp();
  if(load&&active)void loadMap();
}

document.querySelectorAll('.layer').forEach(btn=>btn.onclick=()=>{
  if(btn.disabled)return;
  selectLayer(btn.dataset.layer);
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
  return ({
    final:'最终地形',
    raw:'Raw 地形',
    skeleton:'海陆骨架',
    coast:'海岸约束',
    rivers:'河网',
    lakes:'湖泊',
    water:'最终水体',
    biome:'群系',
    landform:'地貌类型',
    surface:'地表方案',
    profile:'配置置信度',
    anomalies:'最终异常'
  })[v]||v;
}

function updateLayerHelp(){
  const help={
    final:'最终地形：显示 Runtime 的最终地面和水体；生成失败时不会把旧地图冒充成新结果。',
    raw:'Raw 地形：海岸校正前的正式连续高度场，不叠加海洋、河流或湖泊水体。仅刚生成世界保留这份中间模型，用来和最终结果对照。',
    skeleton:'海陆骨架：绿=陆地，深蓝=海洋，浅色边线=海岸，棕=封闭且低于海平面的陆地洼地。',
    coast:'海岸约束：橙=Raw 海陆冲突，黄=需要抬升，蓝=需要削低，粉=应用校正后仍有冲突（理想应为 0）。仅刚生成世界可用。',
    rivers:'河网：直接绘制正式 Waterbed 河段；线宽为验收可视化宽度，不代表地图像素就是实际河宽。',
    lakes:'湖泊：直接绘制正式 Waterbed 湖泊节点和影响半径，便于检查湖泊分布与聚集区域。',
    water:'最终水体：按 Runtime 逐点显示海洋、湖泊和河流的最终水体判定。',
    biome:'群系：显示最终 Minecraft 生物群系家族；用于检查气候/生态区域是否出现不自然的直线或矩形切割。',
    landform:'地貌类型：直接显示 Runtime 的 Plain / Hills / Foothills / Mountain / Plateau / Floodplain 等地貌分类，用来检查平原、丘陵、山麓和山地是否形成合理连续区域。',
    surface:'地表方案：显示最终 SurfaceProfile。沙漠沙地、森林土壤、湿地、Alpine Rock / Alpine Snow 等会分别着色，便于不进游戏就检查大片沙砾和山顶雪线。',
    profile:'配置置信度：颜色越异常越需要检查；语义 fallback / mismatch 会高亮。',
    anomalies:'最终异常：红=海岸冲突，橙=河床逆坡，青=湖床违规，黄=Runtime 回退，紫=Runtime 语义冲突。没有标记表示当前最终审计为 0。'
  };
  $('layerHelp').textContent=help[layer]||'';
}

$('mapZoomIn').onclick=()=>zoomMapCentered(1.25);
$('mapZoomOut').onclick=()=>zoomMapCentered(0.8);
$('mapZoomReset').onclick=resetMapView;

$('mapWrap').addEventListener('wheel',e=>{
  if(!active||$('mapWrap').classList.contains('empty'))return;
  e.preventDefault();
  zoomMapAt(
    e.clientX,
    e.clientY,
    mapZoom*(e.deltaY<0?1.25:0.8)
  );
},{passive:false});

$('map').addEventListener('pointerdown',e=>{
  if(mapZoom<=1.001||e.button!==0)return;
  mapDragging=true;
  mapDragMoved=false;
  mapDragStartX=e.clientX;
  mapDragStartY=e.clientY;
  mapPanStartX=mapPanX;
  mapPanStartY=mapPanY;
  $('mapWrap').classList.add('dragging');
  $('map').setPointerCapture(e.pointerId);
  e.preventDefault();
});

$('map').addEventListener('pointermove',e=>{
  if(!mapDragging)return;
  const dx=e.clientX-mapDragStartX;
  const dy=e.clientY-mapDragStartY;
  if(Math.hypot(dx,dy)>3)mapDragMoved=true;
  mapPanX=mapPanStartX+dx;
  mapPanY=mapPanStartY+dy;
  applyMapTransform();
});

function finishMapDrag(e){
  if(!mapDragging)return;
  mapDragging=false;
  $('mapWrap').classList.remove('dragging');
  try{$('map').releasePointerCapture(e.pointerId)}catch(_){}
  if(mapDragMoved){
    suppressMapClick=true;
    setTimeout(()=>{suppressMapClick=false},0);
  }
  updateActionState();
}

$('map').addEventListener('pointerup',finishMapDrag);
$('map').addEventListener('pointercancel',finishMapDrag);

$('map').onclick=async e=>{
  if(suppressMapClick){
    suppressMapClick=false;
    return;
  }
  if(!active)return;
  const rect=$('map').getBoundingClientRect();
  const fx=Math.max(0,Math.min(1,(e.clientX-rect.left)/rect.width));
  const fz=Math.max(0,Math.min(1,(e.clientY-rect.top)/rect.height));
  const g=active.geometry;
  const x=g.OriginX+fx*g.SpanXBlocks;
  const z=g.OriginZ+fz*g.SpanZBlocks;
  await focusWorldPoint(x,z);
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
  clearExportResult();
  $('modelMessage').textContent='';
  try{
    active=await api('/api/load',{method:'POST',body:JSON.stringify({name})});
    resetMapView();
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
    syncContinentPreset();
    if(!active.intermediateDiagnostics&&(layer==='raw'||layer==='coast')){
      selectLayer('final',{load:false});
    }
    $('topStatus').textContent='已重开';
    $('progressText').textContent=`模型重开完成 · ${active.durationMs}ms`;
    $('modelName').value=name;
    modelNameTouched=true;
    if(!exportNameTouched||!$('exportName').value.trim()){
      $('exportName').value=name+' QA';
      exportNameTouched=false;
    }
    $('modelMessage').innerHTML='<span class="warn">当前是已保存模型的只读重开结果；如需再次保存，请重新生成世界。</span>';
    clearSample();
    renderStats();
    await refreshAnomalies();
    await loadMap();
  }catch(e){
    $('modelMessage').innerHTML='<span class="error">'+escapeHtml(e.message)+'</span>';
  }finally{
    setBusy(false);
  }
};

$('exportMinecraft').onclick=async()=>{
  if(!active)return;
  setBusy(true);
  lastExport=null;
  $('openExport').hidden=true;
  $('exportMessage').innerHTML='<span class="warn">正在编译 Java writer 并导出存档…</span>';
  $('topStatus').textContent='导出中';
  try{
    const radius=Math.max(0,Math.min(31,Number($('exportRadius').value)||0));
    $('exportRadius').value=String(radius);
    const result=await api('/api/export-minecraft',{
      method:'POST',
      body:JSON.stringify({
        name:$('exportName').value,
        directory:$('exportDir').value,
        radius,
        qaMode:$('exportQA').checked,
        overwrite:$('exportOverwrite').checked
      })
    });
    lastExport=result;
    $('openExport').hidden=false;
    $('topStatus').textContent='已导出';
    $('exportMessage').innerHTML=
      '<span class="ok">导出完成</span><br>'+
      'Minecraft '+escapeHtml(result.minecraftVersion)+
      ' · '+Number(result.chunks).toLocaleString()+' Chunk'+
      ' · '+Number(result.regions).toLocaleString()+' Region<br>'+
      '出生点 '+Number(result.spawnX)+', '+Number(result.spawnY)+', '+Number(result.spawnZ)+
      (result.qaMode?' · QA 创造/作弊':' · 生存模式')+
      '<br><span class="muted">'+escapeHtml(result.output)+'</span>';
    renderExportOverlay();
  }catch(e){
    $('topStatus').textContent='导出失败';
    $('exportMessage').innerHTML='<span class="error">'+escapeHtml(e.message)+'</span>';
  }finally{
    setBusy(false);
  }
};

$('openExport').onclick=async()=>{
  if(!lastExport)return;
  try{
    await api('/api/open-folder',{
      method:'POST',
      body:JSON.stringify({path:lastExport.output})
    });
  }catch(e){
    $('exportMessage').innerHTML+='<br><span class="error">'+escapeHtml(e.message)+'</span>';
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


function renderEmptyAnomalies(){
  $('anomalySummary').textContent='生成或重开模型后显示最终异常。';
  $('anomalyList').innerHTML='';
}

async function refreshAnomalies(){
  if(!active){
    renderEmptyAnomalies();
    return;
  }
  try{
    const report=await api('/api/anomalies');
    const counts=report.counts||{};
    $('anomalySummary').innerHTML=
      '共 <b>'+Number(report.total||0)+'</b> 个最终异常 · 海岸 '+Number(counts.coast_mismatch||0)+
      ' · 河床 '+Number(counts.river_upslope||0)+
      ' · 湖床 '+Number(counts.lake_bed||0)+
      ' · Runtime 回退 '+Number(counts.runtime_fallback||0)+
      ' · Runtime 冲突 '+Number(counts.runtime_mismatch||0);
    if(!report.points||report.points.length===0){
      $('anomalyList').innerHTML='<div class="anomaly-empty">当前最终异常为 0</div>';
      return;
    }
    $('anomalyList').innerHTML=report.points.map((p,i)=>
      '<button type="button" class="anomaly-item" data-anomaly-index="'+i+'">'+
        '<strong>'+escapeHtml(p.label)+'</strong>'+
        '<span>坐标 '+Math.round(p.x)+', '+Math.round(p.z)+(p.detail?' · '+escapeHtml(p.detail):'')+'</span>'+
      '</button>'
    ).join('')+(report.truncated?'<div class="anomaly-truncated">异常过多，只显示前 '+report.points.length+' 个位置；总数仍按完整审计统计。</div>':'');
    document.querySelectorAll('.anomaly-item').forEach((btn)=>{
      btn.onclick=async()=>{
        const p=report.points[Number(btn.dataset.anomalyIndex)];
        if(!p)return;
        selectLayer('anomalies');
        await focusWorldPoint(p.x,p.z);
      };
    });
  }catch(e){
    $('anomalySummary').innerHTML='<span class="error">'+escapeHtml(e.message)+'</span>';
    $('anomalyList').innerHTML='';
  }
}

async function focusWorldPoint(x,z){
  if(!active)return;
  lastFocusPoint={x,z};
  renderCursorOverlay();
  try{
    renderSample(await api('/api/sample?x='+encodeURIComponent(x)+'&z='+encodeURIComponent(z)));
  }catch(err){
    $('sampleEmpty').hidden=true;
    $('sample').hidden=false;
    $('sample').innerHTML='<div class="error-card">'+escapeHtml(err.message)+'</div>';
  }
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
    ${diagnosticKV('最终海岸冲突',active.finalCoastMismatch)}
    ${diagnosticKV('最终河床逆坡',active.finalRiverUpslope)}
    ${diagnosticKV('最终湖床违规',active.finalLakeBedViolations)}
    ${diagnosticKV('Runtime 节点回退',active.runtimeFallback)}
    ${diagnosticKV('Runtime 节点冲突',active.runtimeMismatch)}
    ${active.intermediateDiagnostics?`
      ${kv('海岸校正节点',active.coastAdjustedNodes)}
      ${kv('Raw 海陆冲突',active.surfaceMismatchBefore)}
      ${kv('约束后海陆冲突',active.surfaceMismatchAfter)}
      ${kv('应用后海陆冲突',active.appliedCoastSurfaceMismatch)}
      ${kv('Raw 河流逆坡',active.rawUpslopeRiverEdges)}
      ${kv('目标河流逆坡',active.targetUpslopeRiverEdges)}
      ${kv('海岸后湖床待挖',active.lakeBedViolations)}
    `:''}
  </div>`;
}

function clearSample(){
  lastFocusPoint=null;
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
function diagnosticKV(k,v){
  const n=Number(v)||0;
  return kv(k,n===0?'<span class="ok">0</span>':'<span class="error">'+n+'</span>');
}
function f(v){return Number(v).toFixed(6)}
function formatBytes(n){return n<1048576?(n/1024).toFixed(1)+' KB':(n/1048576).toFixed(2)+' MB'}
function escapeHtml(s){return String(s).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]))}
function escapeAttr(s){return escapeHtml(s)}

$('map').addEventListener('load',()=>{
  renderExportOverlay();
  renderCursorOverlay();
  updateActionState();
});
window.addEventListener('resize',()=>{
  renderExportOverlay();
  renderCursorOverlay();
});
window.addEventListener('beforeunload',()=>{if(mapObjectURL)URL.revokeObjectURL(mapObjectURL)});

updateRangePreview();
updateContinentLabels();
syncContinentPreset();
updateLayerHelp();
updateActionState();
renderEmptyAnomalies();
refreshModels();
refreshStatus();
setInterval(()=>{if(!poll)void refreshStatus()},2000);
