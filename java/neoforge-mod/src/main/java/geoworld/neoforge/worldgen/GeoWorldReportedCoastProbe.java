package geoworld.neoforge.worldgen;

import geoworld.neoforge.creation.GeoWorldCreationConfig;
import geoworld.neoforge.engine.GeoWorldEngineBridge;
import java.util.Map;
import java.util.TreeMap;

/** Small, reproducible, real-column audit of user-provided lake/sea sites. */
public final class GeoWorldReportedCoastProbe {
    private GeoWorldReportedCoastProbe() {}

    public static void main(String[] args) throws Exception {
        long seed = -563659577384754347L;
        var model = GeoWorldEngineBridge.generate(seed, GeoWorldCreationConfig.defaults()).model();
        int[][] sites = {{-6289, 3374}, {-6586, 4090}};
        for (int[] site : sites) {
            int x0=site[0], z0=site[1];
            int leftChunk = Math.floorDiv(x0-48,16);
            int rightChunk = Math.floorDiv(x0+48,16);
            int topChunk = Math.floorDiv(z0-48,16);
            int bottomChunk = Math.floorDiv(z0+48,16);
            Map<Long, GeoWorldShoreline.Column> cells = new java.util.HashMap<>();
            for (int cz=topChunk;cz<=bottomChunk;cz++) {
                for (int cx=leftChunk;cx<=rightChunk;cx++) {
                    var plan=GeoWorldShoreline.planChunk(model,cx*16,cz*16);
                    for (int z=0;z<16;z++) for(int x=0;x<16;x++)
                        cells.put(key(cx*16+x,cz*16+z),plan.column(x,z));
                }
            }
            int[] distanceCount=new int[11],heightDrop=new int[11];
            int water=0, land=0, tapered=0, cappedSlope=0, maxGround=0;
            Map<String,Integer> kinds=new TreeMap<>(),terrain=new TreeMap<>();
            int eligible=0;
            for(int z=z0-40;z<=z0+40;z++)
              for(int x=x0-40;x<=x0+40;x++) {
                var c=cells.get(key(x,z));if(c==null)continue;
                terrain.merge(c.sample().terrainVariant(),1,Integer::sum);
                if(c.waterBoundary()>c.groundBoundary()){water++;kinds.merge(c.shoreWaterKind(),1,Integer::sum);}
                else land++;
                int d=c.shoreDistance();
                if(d>=1&&d<=10&&("Ocean".equals(c.shoreWaterKind())||"Lake".equals(c.shoreWaterKind()))){
                    distanceCount[d]++;
                    int drop=c.originalGroundBoundary()-c.groundBoundary();
                    heightDrop[d]+=drop;
                    if(drop>0){tapered++;maxGround=Math.max(maxGround,drop);}
                    if(c.originalGroundBoundary()-c.shoreWaterBoundary()>=4&&c.originalGroundBoundary()-c.shoreWaterBoundary()<=32)eligible++;
                }
                for (int[]dir:new int[][]{{1,0},{0,1}}) {
                    var n=cells.get(key(x+dir[0],z+dir[1]));
                    if(n!=null && c.waterBoundary()==c.groundBoundary() &&
                      n.waterBoundary()==n.groundBoundary() &&
                      c.shoreDistance()<=10 && c.shoreDistance()>0 &&
                      Math.abs(c.groundBoundary()-n.groundBoundary())>=3) cappedSlope++;
                }
              }
            System.out.printf("COAST site=(%d,%d) land=%d water=%d waterKinds=%s terrains=%s candidates=%d altered=%d maxDrop=%d steepDryEdges=%d%n",
                x0,z0,land,water,kinds,terrain,eligible,tapered,maxGround,cappedSlope);
            for(int d=1;d<=10;d++)if(distanceCount[d]>0)
                System.out.printf("  d=%d columns=%d avgLowered=%.2f%n",d,distanceCount[d],(double)heightDrop[d]/distanceCount[d]);
            if (Boolean.getBoolean("geoworld.coastProbeVerbose")) {
              for(int z : new int[]{z0-8,z0,z0+8}) {
                System.out.println("TRANSSECT z="+z+" x height/water/depth/d");
                for(int x=x0-48;x<=x0+48;x+=3) {
                    var c=cells.get(key(x,z));
                    if(c==null)continue;
                    int depth=c.waterBoundary()-c.groundBoundary();
                    System.out.printf(" %d:%d/%d/%d/%d:%s",x,c.groundBoundary(),c.shoreWaterBoundary(),
                        depth,c.shoreDistance(),c.shoreWaterKind());
                }
                System.out.println();
            }
            }
            int steepSeabed=0,coastalDeep=0,landRaised=0,waterNearShore=0;
            int wetBedRaised=0, wetDryCliffs=0, maxWetDryCliff=0;
            for(int z=z0-40;z<=z0+40;z++)
                for(int x=x0-40;x<=x0+40;x++){
                    var c=cells.get(key(x,z));
                    if(c.waterBoundary()>c.groundBoundary() &&
                       c.groundBoundary()>c.originalGroundBoundary())wetBedRaised++;
                    if(c.waterBoundary()<=c.groundBoundary()) {
                        if(c.groundBoundary()>c.originalGroundBoundary())landRaised++;
                        continue;
                    }
                    if(!"Ocean".equals(c.shoreWaterKind()) &&
                       !"Lake".equals(c.shoreWaterKind()))continue;
                    boolean shoreAdjacent=false;
                    for(int dz=-1;dz<=1;dz++)for(int dx=-1;dx<=1;dx++){
                        if(dx==0&&dz==0)continue;
                        var n=cells.get(key(x+dx,z+dz));
                        if(n!=null&&n.waterBoundary()==n.groundBoundary())shoreAdjacent=true;
                    }
                    if(shoreAdjacent){
                        waterNearShore++;
                        int depth=c.waterBoundary()-c.groundBoundary();
                        if(depth>=4)coastalDeep++;
                    }
                    for(int[] dir :new int[][]{{1,0},{0,1},{-1,0},{0,-1}}){
                        var n=cells.get(key(x+dir[0],z+dir[1]));
                        if(n==null)continue;
                        if(n.waterBoundary()>n.groundBoundary() &&
                            Math.abs(c.groundBoundary()-n.groundBoundary())>2)steepSeabed++;
                        if(n.waterBoundary()<=n.groundBoundary()){
                            int difference=n.groundBoundary()-c.groundBoundary();
                            maxWetDryCliff=Math.max(maxWetDryCliff,difference);
                            if(difference>2)wetDryCliffs++;
                        }
                    }
                }
            System.out.printf("ADDITIONAL steepSeabed=%d waterAdjacentDry=%d depth>=4=%d raisedDry=%d raisedWetBed=%d wetDryCliffs=%d maxCliff=%d%n",
                steepSeabed,waterNearShore,coastalDeep,landRaised,wetBedRaised,wetDryCliffs,maxWetDryCliff);
            if (site == sites[0] && (wetBedRaised < 60 || wetDryCliffs != 0)) {
                throw new AssertionError("reported sand sea-wall not repaired: "+wetBedRaised+" raised seabed cells, "+wetDryCliffs+" steep wet/dry cliffs");
            }
            if (Boolean.getBoolean("geoworld.coastProbeVerbose")) {
            for(int z=z0-40;z<=z0+40;z+=4){
                StringBuilder row=new StringBuilder();
                for(int x=x0-40;x<=x0+40;x+=4){
                  var c=cells.get(key(x,z));
                  if(c.waterBoundary()>c.groundBoundary())row.append(
                    "Ocean".equals(c.shoreWaterKind())?'~':
                    "Lake".equals(c.shoreWaterKind())?'L':
                    "River".equals(c.shoreWaterKind())?'R':'W');
                  else if(c.originalGroundBoundary()>c.groundBoundary())row.append('v');
                  else {int h=c.groundBoundary()-63;row.append(h<0?'_':h<10?(char)('0'+h):h<36?(char)('A'+h-10):'#');}
                }
                System.out.printf("%5d %s%n",z,row);
            }
            }
        }
    }

    private static long key(int x, int z) {
        return ((long)x<<32)|(z&0xffffffffL);
    }
}
