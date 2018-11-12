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
	mKey := prototype.MakeAccountName("pbTool")
	wrap := table.NewSoDemoWrap(db, mKey)
	if wrap == nil {
		//crreate fail , the db already contain table with current mainKey
		log.Println("crreate fail , the db already contain table with current mainKey")
		return
	}

	//2.create the pb struct
	data := table.SoDemo{
	 	Owner:mKey,
	 	Title:"hello",
	 	Content:"test the pb tool",
	 	Idx: 1000,
	 	LikeCount:1,
	 	Taglist:"#NBA",
	 	ReplayCount:100,
		PostTime:prototype.MakeTimeSecondPoint(20120401),
	 }

	 //3.save table data to db
	 res := wrap.CreateDemo(&data)
	 if !res {
	 	 log.Fatalln("create new table of Demo faile")
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
		log.Printf("modify tilte fail")
	}
	//modify title
	tMdRes := wrap.MdContent("hello world")
	if !tMdRes {
		log.Println("modify tilte fail")
	}


	/*
	  --------------------------
	   Modify property value (******can't modify the mainkey)
	  --------------------------*/
	//modify content
	cMdRes := wrap.MdContent("test md the content")
	if !cMdRes {
		log.Printf("modify content fail")
	}


	/*--------------------------
	   Sort Query List
	  --------------------------*/
     //1.create the sort wrap for property which is surpport sort (E.g postTime)
	 tSortWrap := table.SDemoPostTimeWrap{}
	tSortWrap.Dba = db
	 //2.start query data of range(sort by order)
	 iter := tSortWrap.QueryListByOrder(*prototype.MakeTimeSecondPoint(20120401),
	 	*prototype.MakeTimeSecondPoint(20120401))
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
			}
		}

	 }else {
	 	fmt.Println("there is no data exist in range")
	 }
	 //query by reverse order
	iter1 := tSortWrap.QueryListByOrder(*prototype.MakeTimeSecondPoint(20136688),
		*prototype.MakeTimeSecondPoint(20136666))
	//we can get the main key and sub key by the returned iterator
	//if query by reverse order the start value can't less than end value
	if iter1 != nil {
		for iter1.Next() {
			mKeyPtr := tSortWrap.GetMainVal(iter)
			if mKeyPtr == nil {
				fmt.Println("query by reverse order get main key fail")
			}
			mSubPtr := tSortWrap.GetSubVal(iter1)
			if mSubPtr == nil {
				fmt.Println("query by reverse order get postTime fail")
			}
		}

	}else {
		fmt.Println("there is no data exist in range1")
	}

     //query single value but not a range,start and end set the same value
	iter2 := tSortWrap.QueryListByOrder(*prototype.MakeTimeSecondPoint(20136666),
		*prototype.MakeTimeSecondPoint(20136666))
	if iter2 != nil {
		mKeyPtr := tSortWrap.GetMainVal(iter2)
		if mKeyPtr == nil {
			fmt.Println("get main key fail")
		}
	}

	/*
	 --------------------------
	  unique Query List (only support query the property which is flag unique)
	 --------------------------*/
	 //1.create the uni wrap of property which is need unique query
	 var idx int64 = 1000
	 //create the UniXXXWrap
	 uniWrap := table.UniDemoIdxWrap{}
	 //set the dataBase to UniXXXWrap
	 uniWrap.Dba = db
	 //2.use UniQueryXX func to query data meanWhile return the table wrap
	 dWrap := uniWrap.UniQueryIdx(&idx)
	 if dWrap == nil {
	 	fmt.Printf("uni query fail")
	 }else {
		 title := dWrap.GetTitle()
		 fmt.Printf("the title of index is %s",title)
	 }

	//unique query mainkey(E.g query owner)
	mUniWrap := table.UniDemoOwnerWrap{}
	mUniWrap.Dba = db
	wrap1 := mUniWrap.UniQueryOwner(prototype.MakeAccountName("test"))
	if wrap1 != nil {
		fmt.Printf("owner is test,the idx is %d",wrap1.GetIdx())
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
