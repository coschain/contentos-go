#!/bin/bash

work_path=$(dirname $(readlink -f $0))
${work_path}/bin/cosd init
exec ${work_path}/bin/cosd start