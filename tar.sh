#! /bin/bash
filename=`date +%Y%m%d-%H%M`
tar zcvf GwUpinsUser_$filename.tar.gz \
DBConfig.yml \
Env.yml \
CheckEqIrankZero.txt \
CheckOldData.txt \
GwUpinsUser
