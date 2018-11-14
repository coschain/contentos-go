package main

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/db/storage"
	"log"
)

func main() {
	/*
	  ------------------------------------------------
	  type,pName,mKey,unique,sort,reverseSort
	  **the csv file Field  Explanation
	  type: the data type of property in table
	  pName: the name of property in table
	  mKey: whether the property is a primary key
	  unique: whether the property is support unique query (0:not support 1:support)
	  sort: whether the property is support sort by order when query (0:not support 1:support)
	  reverseSort:whether the property is support sort by reverse order when query (0:not support 1:support)
	  ------------------------------------------------
	*/

	//db, _ := storage.NewLevelDatabase("/Users/yykingking/abc123.db")
	db,err := storage.NewDatabase("./demos/pbTool.db")
	if err != nil {
		return
	}
	db.Start(nil)
	//defer db.Close()

	//1.create the table wrap
	//we can use the type  which is contained in another created pb struct,
	// such as "prototype.account_name" in AccountName 、prototype.time_point_sec
	//MakeXXX func can create a pb struct
	mKey := prototype.MakeAccountName("myName")
	wrap := table.NewSoDemoWrap(db, mKey)
	if wrap == nil {
		//crreate fail , the db already contain table with current mainKey
		log.Println("crreate fail , the db already contain table with current mainKey")
		return
	}
    if wrap.CheckExist() {
    	wrap.RemoveDemo()
	}
	//2.create the pb struct
	data := table.SoDemo{
	 	Owner:mKey,
	 	Title:"hello",
	 	Content:"test the pb tool",
	 	Idx: 1001,
	 	LikeCount:100,
	 	Taglist:"#NBA",
	 	ReplayCount:100,
		PostTime:creTimeSecondPoint(20120401),
	 }

	 //3.save table data to db
	 res := wrap.CreateDemo(&data)
	 if !res {
		 fmt.Println("create new table of Demo fail")
		 return
	 }

	 /*
	   --------------------------
	   Get Property（the GetXXX function  return the property value）
	   --------------------------*/

	 //get title
	 t := wrap.GetTitle()
	 if t != "" {
		 fmt.Printf("the title is %s \n",t)
	 }else {
		 fmt.Printf("get title fail")
	 }

	 //get content
	 c := wrap.GetContent()
	if c != "" {
		fmt.Printf("the content is %s \n",c)
	}else {
		fmt.Println("modify tilte fail")
	}
	//modify title
	tMdRes := wrap.MdContent("hello world")
	if !tMdRes {
		fmt.Println("modify tilte fail")
	}


	/*
	  --------------------------
	   Modify property value (******can't modify the mainkey)
	  --------------------------*/
	//modify content
	cMdRes := wrap.MdContent("test md the content")
	if !cMdRes {
		fmt.Println("modify content fail")
	}


	/*--------------------------
	   Sort Query List
	  --------------------------*/
     //1.create the sort wrap for property which is surpport sort (E.g postTime)
	 tSortWrap := table.SDemoPostTimeWrap{}
	tSortWrap.Dba = db
	 //2.start query data of range(sort by order)
	//start = nil  end = nil (query the db from start to end)
	//start = nil (query from start the db)
	//end = nil (query to the end of db)
	iter := tSortWrap.QueryListByOrder(creTimeSecondPoint(20120401),
		creTimeSecondPoint(20120415))
	//we can get the main key and sub key by the returned iterator
	//if query by order the start value can't greater than end value
	if iter != nil {
		for iter.Next() {
			//get the mainkey value (GetMainVal return the ptr of value)

			mKeyPtr := tSortWrap.GetMainVal(iter)
			if mKeyPtr == nil {
				fmt.Println("get main key fail")
			}else {
				fmt.Printf("the main key is %s in range \n",mKeyPtr.Value)
			}
			//get subKey value (the postTime value)
			mSubPtr := tSortWrap.GetSubVal(iter)
			if mSubPtr == nil {
				fmt.Println("get postTime fail")
			}else {
				fmt.Printf("the postTime is %d \n",mSubPtr.UtcSeconds)
			}
		}
		//******* we must delete the iterator after end of use,otherwise maybe cause unKnow error *******//
		tSortWrap.DelIterater(iter)
	}else {
		fmt.Println("there is no data exist in range posttime")
	}
	//query by reverse order
	//start = nil  end = nil (query the db from start to end)
	//start = nil (query from start the db)
	//end = nil (query to the end of db)
	iter1 := tSortWrap.QueryListByRevOrder(creTimeSecondPoint(20120415),
		creTimeSecondPoint(20120401))
	//we can get the main key and sub key by the returned iterator
	//if query by reverse order the start value can't less than end value
	if iter1 != nil {
		for iter1.Next() {
			mKeyPtr := tSortWrap.GetMainVal(iter1)
			if mKeyPtr == nil {
				fmt.Println("query by reverse order get main key fail")
			}else {
				fmt.Printf("the main key is %s in reverse order  \n",mKeyPtr.Value)
			}
			mSubPtr := tSortWrap.GetSubVal(iter1)
			if mSubPtr == nil {
				fmt.Println("query by reverse order get postTime fail")
			}else {
				fmt.Printf("the postTime is %d in reverse order \n",mSubPtr.UtcSeconds)
			}
		}
     //******** delete the iterator ***********//
		tSortWrap.DelIterater(iter1)
	}else {
		fmt.Println("there is no data exist in reverse order")
	}


    //query single value but not a range,start and end set the same value
	iter2 := tSortWrap.QueryListByOrder(creTimeSecondPoint(20136666),
		creTimeSecondPoint(20136666))
	if iter2 != nil {
		if iter2.Next() {
			mKeyPtr := tSortWrap.GetMainVal(iter2)
			if mKeyPtr == nil {
				fmt.Println("get main key fail in range")
			}
		}

		tSortWrap.DelIterater(iter2)
	}

	//query without start
	iter3 := tSortWrap.QueryListByOrder(nil,creTimeSecondPoint(20120422))
	if iter3 != nil  {
		for iter3.Next() {
			mKeyPtr := tSortWrap.GetMainVal(iter3)
			if mKeyPtr == nil {
				fmt.Println("get main key fail in range when query without start 1111")
			}else {
				fmt.Printf("the main key is %s in range when query without start  \n",mKeyPtr.Value)
			}
		}
		tSortWrap.DelIterater(iter3)
	}else {
		fmt.Println("there is no data exist without start")
	}

	//query without end
	iter4 := tSortWrap.QueryListByOrder(creTimeSecondPoint(20120000),nil)
	if iter4 != nil  {
		for iter4.Next() {
			mKeyPtr := tSortWrap.GetMainVal(iter4)
			if mKeyPtr == nil {
				fmt.Println("get main key fail in range when query without end")
			}else {
				fmt.Printf("the main key is %s in range when query without end \n",mKeyPtr.Value)
			}
			tSortWrap.DelIterater(iter4)
		}

	}else {
		fmt.Println("there is no data in range when query without end")
	}

	//query without start and end
	iter5 := tSortWrap.QueryListByOrder(nil,nil)
	if iter5 != nil {
		for iter5.Next() {
			mKeyPtr := tSortWrap.GetMainVal(iter5)
			if mKeyPtr == nil {
				fmt.Println("get main key fail in range when query without start and end")
			} else {
				fmt.Printf("the main key is %s when query without start and end  \n", mKeyPtr.Value)
			}
		}
		tSortWrap.DelIterater(iter5)
	}

	//query without start and end by reverse order
	iter6 := tSortWrap.QueryListByRevOrder(nil,nil)
	if iter6 != nil {
		for iter6.Next() {
			mKeyPtr := tSortWrap.GetMainVal(iter6)
			if mKeyPtr == nil {
				fmt.Println("get main key fail in range when query without start and end by reverse sort ")
			}else {
				fmt.Printf("the main key is %s in range when query without start and end by reverse sort \n",mKeyPtr.Value)
			}
		}
		tSortWrap.DelIterater(iter6)
	}else {
		fmt.Println("there is no data in reverse order without start and end")
	}

	/*
	 --------------------------
	  unique Query List (only support query the property which is flag unique)
	 --------------------------*/
	 //1.create the uni wrap of property which is need unique query
	 var idx int64 = 1001
	 //create the UniXXXWrap
	 uniWrap := table.UniDemoIdxWrap{}
	 //set the dataBase to UniXXXWrap
	 uniWrap.Dba = db
	 //2.use UniQueryXX func to query data meanWhile return the table wrap
	 dWrap := uniWrap.UniQueryIdx(&idx)
	 if dWrap == nil {
	 	fmt.Printf("uni query fail \n")
	 }else {
		 title := dWrap.GetTitle()
		 fmt.Printf("the title of index is %s \n",title)
	 }

	//unique query mainkey(E.g query owner)
	mUniWrap := table.UniDemoOwnerWrap{}
	mUniWrap.Dba = db
	wrap1 := mUniWrap.UniQueryOwner(prototype.MakeAccountName("myName"))
	if wrap1 != nil {
		fmt.Printf("owner is test,the idx is %d \n",wrap1.GetIdx())
	}

	  /*
	    remove tabale data from db
	  */
	  //judge the table of current mainKey if is exist
	  isExsit := wrap.CheckExist()
	  if isExsit {
	  	 res := wrap.RemoveDemo()
	  	 if !res {
	  	 	fmt.Println("remove the table data fail")
		 }
	  }

	 db.Close()
}

func creTimeSecondPoint(t uint32) *prototype.TimePointSec {
	val := prototype.TimePointSec{UtcSeconds:t}
	return &val
}